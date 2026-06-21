package infrastructure

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/elkasir/api/internal/platform/config"
)

func TestMidtransIsPaid(t *testing.T) {
	for _, s := range []string{"settlement", "Settlement", "capture"} {
		if !mtIsPaid(s) {
			t.Errorf("%q seharusnya dianggap lunas", s)
		}
	}
	for _, s := range []string{"pending", "deny", "expire", "cancel", ""} {
		if mtIsPaid(s) {
			t.Errorf("%q seharusnya TIDAK lunas", s)
		}
	}
}

func TestMidtransFraudAccepted(t *testing.T) {
	for _, s := range []string{"", "accept", "ACCEPT"} {
		if !mtFraudAccepted(s) {
			t.Errorf("fraud_status %q seharusnya lolos", s)
		}
	}
	for _, s := range []string{"deny", "challenge"} {
		if mtFraudAccepted(s) {
			t.Errorf("fraud_status %q seharusnya ditolak", s)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", "abc", "def"); got != "abc" {
		t.Errorf("firstNonEmpty = %q, want abc", got)
	}
	if got := firstNonEmpty("", ":"); got != "" {
		t.Errorf("firstNonEmpty hanya ':' harus kosong, got %q", got)
	}
}

func TestMidtransVerifyWebhook(t *testing.T) {
	g := newMidtrans(config.Midtrans{ServerKey: "srv-key"})
	sum := sha512.Sum512([]byte("ORDER-1" + "200" + "12000.00" + "srv-key"))
	sig := hex.EncodeToString(sum[:])
	body := []byte(`{"order_id":"ORDER-1","status_code":"200","gross_amount":"12000.00","signature_key":"` + sig + `","transaction_status":"settlement"}`)

	if !g.verifyWebhook(nil, body) {
		t.Fatal("signature Midtrans valid seharusnya lolos")
	}
	bad := []byte(`{"order_id":"ORDER-1","status_code":"200","gross_amount":"12000.00","signature_key":"deadbeef","transaction_status":"settlement"}`)
	if g.verifyWebhook(nil, bad) {
		t.Fatal("signature Midtrans salah seharusnya ditolak")
	}

	ev, err := g.parseWebhook(body)
	if err != nil || ev.OrderRef != "ORDER-1" || !ev.Paid {
		t.Fatalf("parseWebhook midtrans = %+v err=%v, want OrderRef=ORDER-1 Paid=true", ev, err)
	}
}

func TestTripayQRISFeeFallback(t *testing.T) {
	cases := []struct{ amount, want int64 }{
		{0, 0},
		{100_000, 1_450},  // 750 + 700 (0.7%)
		{113_500, 1_545},  // 750 + ceil(794.5)=795 — dibulatkan ke atas
		{113_000, 1_541},  // 750 + 791
	}
	for _, c := range cases {
		if got := tripayQRISFeeFallback(c.amount); got != c.want {
			t.Errorf("tripayQRISFeeFallback(%d)=%d want %d", c.amount, got, c.want)
		}
	}
}

func TestTripayVerifyAndParseWebhook(t *testing.T) {
	g := newTripay(config.Tripay{APIKey: "k", PrivateKey: "priv", MerchantCode: "T0001"}, "http://x/api/v1/webhooks/payment")
	body := []byte(`{"reference":"T123","merchant_ref":"ORDER-9","status":"PAID","total_amount":12000}`)

	mac := hmac.New(sha256.New, []byte("priv"))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	h := http.Header{}
	h.Set("X-Callback-Signature", sig)
	h.Set("X-Callback-Event", "payment_status")
	if !g.verifyWebhook(h, body) {
		t.Fatal("signature Tripay valid seharusnya lolos")
	}

	wrong := http.Header{}
	wrong.Set("X-Callback-Signature", "deadbeef")
	if g.verifyWebhook(wrong, body) {
		t.Fatal("signature Tripay salah seharusnya ditolak")
	}

	ev, err := g.parseWebhook(body)
	if err != nil || ev.OrderRef != "ORDER-9" || !ev.Paid {
		t.Fatalf("parseWebhook tripay = %+v err=%v, want OrderRef=ORDER-9 Paid=true", ev, err)
	}

	unpaid := []byte(`{"reference":"T123","merchant_ref":"ORDER-9","status":"EXPIRED"}`)
	if ev, _ := g.parseWebhook(unpaid); ev.Paid {
		t.Fatal("status EXPIRED seharusnya Paid=false")
	}
}
