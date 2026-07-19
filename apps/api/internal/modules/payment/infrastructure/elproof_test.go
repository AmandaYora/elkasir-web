package infrastructure

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
)

func TestElProofVerifyWebhook(t *testing.T) {
	g := newElproof("app_1", "sekret", "")
	body := []byte(`{"orderRef":"INV-1","paid":true}`)

	mac := hmac.New(sha256.New, []byte("sekret"))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	h := http.Header{}
	h.Set("X-Webhook-Signature", sig)
	if !g.verifyWebhook(h, body) {
		t.Fatal("signature ElProof valid seharusnya lolos")
	}

	// ElProof's header carries the raw hex digest with NO "sha256=" prefix — unlike Elkasir's
	// own X-Elkasir-Signature — so a prefixed value must NOT verify even with the right secret.
	prefixed := http.Header{}
	prefixed.Set("X-Webhook-Signature", "sha256="+sig)
	if g.verifyWebhook(prefixed, body) {
		t.Fatal("signature dengan prefix sha256= seharusnya ditolak (ElProof tidak pakai prefix)")
	}

	wrong := http.Header{}
	wrong.Set("X-Webhook-Signature", "deadbeef")
	if g.verifyWebhook(wrong, body) {
		t.Fatal("signature salah seharusnya ditolak")
	}

	empty := newElproof("app_1", "", "")
	if empty.verifyWebhook(h, body) {
		t.Fatal("gateway tanpa secret (mode simulasi) seharusnya selalu menolak verifikasi")
	}
}

func TestElProofParseWebhook(t *testing.T) {
	g := newElproof("app_1", "sekret", "")

	ev, err := g.parseWebhook([]byte(`{"orderRef":"INV-1","paid":true}`))
	if err != nil {
		t.Fatalf("parseWebhook error: %v", err)
	}
	// ElProof's payload has no eventId — orderRef doubles as EventID (safe: webhook_events'
	// unique key is scoped per-provider, so this never collides with tripay/midtrans events).
	if ev.EventID != "INV-1" || ev.OrderRef != "INV-1" || !ev.Paid {
		t.Fatalf("parseWebhook = %+v, want EventID=OrderRef=INV-1 Paid=true", ev)
	}

	ev2, err := g.parseWebhook([]byte(`{"orderRef":"INV-2","paid":false}`))
	if err != nil || ev2.Paid {
		t.Fatalf("parseWebhook paid=false = %+v err=%v", ev2, err)
	}

	if _, err := g.parseWebhook([]byte(`not json`)); err != paymentclient.ErrInvalidPayload {
		t.Fatalf("parseWebhook body rusak harus ErrInvalidPayload, got %v", err)
	}
}

func TestElProofCreateChargeRejectsUnsupportedChannel(t *testing.T) {
	g := newElproof("app_1", "sekret", "http://unused.invalid")
	// Channel validation happens before any HTTP call — subscription only ever uses QRIS
	// (YAGNI, see elproof.go's createCharge doc), so virtual_account must be rejected locally
	// without wasting a round-trip to ElProof.
	_, err := g.createCharge(context.Background(), "INV-1", 10_000, paymentclient.ChannelVA, paymentclient.ChannelOptions{})
	if err == nil || !strings.Contains(err.Error(), "belum didukung") {
		t.Fatalf("createCharge kanal VA seharusnya ditolak lokal, got err=%v", err)
	}
}

// fakeElProofServer stands in for elproof.elcodelabs.com — serves the three endpoints
// elproof.go actually calls (token exchange, create charge, status check) so createCharge/
// checkStatus/ensureToken can be exercised end to end without a real network dependency.
type fakeElProofServer struct {
	tokenHits int64
	chargeReq map[string]any
}

func newFakeElProofServer(t *testing.T) (*httptest.Server, *fakeElProofServer) {
	f := &fakeElProofServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/app/token", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&f.tokenHits, 1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true, "message": "ok",
			"data": map[string]any{"accessToken": "tok-123", "expiresIn": 3600},
		})
	})
	mux.HandleFunc("/external/payments/charges", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok-123" {
			t.Errorf("createCharge harus kirim Authorization: Bearer tok-123, got %q", got)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.chargeReq = body
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true, "message": "ok",
			"data": map[string]any{
				"orderRef": body["orderRef"], "providerRef": "PRV-1", "channel": "QRIS",
				"qrImageUrl": "https://elproof.example/qr.png", "amount": body["amount"], "status": "unpaid",
			},
		})
	})
	mux.HandleFunc("/external/payments/charges/INV-1/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true, "message": "ok",
			"data": map[string]any{"orderRef": "INV-1", "providerRef": "PRV-1", "status": "paid"},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, f
}

func TestElProofTokenIsCachedAcrossCalls(t *testing.T) {
	srv, fake := newFakeElProofServer(t)
	g := newElproof("app_1", "sekret", srv.URL)

	if _, err := g.ensureToken(context.Background()); err != nil {
		t.Fatalf("ensureToken #1: %v", err)
	}
	if _, err := g.ensureToken(context.Background()); err != nil {
		t.Fatalf("ensureToken #2: %v", err)
	}
	// A fresh 1-hour token must be reused, not re-exchanged — ElProof rate-limits
	// /auth/app/token to 10 req/min, so re-exchanging on every call would blow that budget
	// under normal traffic (see elproof.go's ensureToken doc).
	if got := atomic.LoadInt64(&fake.tokenHits); got != 1 {
		t.Fatalf("token exchange dipanggil %d kali, want 1 (harus di-cache)", got)
	}
}

func TestElProofCreateChargeSuccess(t *testing.T) {
	srv, fake := newFakeElProofServer(t)
	g := newElproof("app_1", "sekret", srv.URL)

	res, err := g.createCharge(context.Background(), "INV-1", 150_000, paymentclient.ChannelQRIS, paymentclient.ChannelOptions{})
	if err != nil {
		t.Fatalf("createCharge: %v", err)
	}
	if res.Ref != "PRV-1" || res.QRImageURL != "https://elproof.example/qr.png" {
		t.Fatalf("createCharge result = %+v, want Ref=PRV-1 QRImageURL set", res)
	}
	// customerName/customerEmail/customerPhone must always be sent — verified empirically
	// against the real ElProof API (see elproof.go's createCharge doc): omitting them causes a
	// generic 500, since ElProof wraps Tripay and Tripay itself requires these fields.
	for _, field := range []string{"customerName", "customerEmail", "customerPhone"} {
		if fake.chargeReq[field] == "" || fake.chargeReq[field] == nil {
			t.Errorf("request charge harus selalu menyertakan %q", field)
		}
	}
}

func TestElProofCheckStatus(t *testing.T) {
	srv, _ := newFakeElProofServer(t)
	g := newElproof("app_1", "sekret", srv.URL)

	status, err := g.checkStatus(context.Background(), "INV-1")
	if err != nil {
		t.Fatalf("checkStatus: %v", err)
	}
	if !status.Paid || status.RawStatus != "paid" {
		t.Fatalf("checkStatus = %+v, want Paid=true RawStatus=paid", status)
	}
}

func TestElProofTokenExchangeRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false, "message": "appId atau secret tidak valid",
			"errors": map[string]string{"code": "unauthorized"},
		})
	}))
	defer srv.Close()

	g := newElproof("app_1", "salah", srv.URL)
	_, err := g.ensureToken(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("ensureToken dengan secret salah harus gagal dengan code unauthorized, got %v", err)
	}
}
