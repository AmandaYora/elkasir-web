// Gateway ElProof (elproof.elcodelabs.com) — dompet TERPISAH yang HANYA dipakai untuk billing
// subscription (paymentclient.AppSubscribe), berdampingan dengan (bukan menggantikan) wallet
// Tripay/Midtrans yang dipakai selforder (PLAN.md §11). Elkasir bertindak sebagai APP EKSTERNAL
// di ElProof (appId "Elkasir-Billing") — bukan lagi provider untuk pihak ketiga (Part 3
// dipensiunkan), melainkan client dari produk gateway terpisah.
//
// Kontrak wire ElProof BERBEDA dari Tripay/Midtrans di beberapa hal penting, diverifikasi dari
// docs/PAYMENT_INTEGRATION_GUIDE.md + docs/postman/ElProof-Payment-Gateway.postman_collection.json:
//   - Auth: client-credentials (POST /auth/app/token → accessToken berlaku 1 jam, TIDAK ADA
//     refresh token) — di-cache di sini, bukan ditukar ulang setiap panggilan.
//   - Status check dikunci ke ORDER REF milik pemanggil (GET /external/payments/charges/
//     {orderRef}/status), BUKAN providerRef seperti Tripay/Midtrans.
//   - Error envelope: `errors` adalah OBJEK `{code}`, bukan array seperti konvensi Elkasir sendiri.
//   - Webhook: header `X-Webhook-Signature` (hex mentah, TANPA prefix "sha256="), payload
//     `{orderRef, paid, amount, paidAt}` — TANPA `eventId`.
package infrastructure

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
)

const elproofDefaultBaseURL = "https://elproof.elcodelabs.com/api/v1"

type elproofGateway struct {
	appID   string
	secret  string
	baseURL string
	http    *http.Client

	tokenMu     sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

func newElproof(appID, secret, baseURL string) *elproofGateway {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = elproofDefaultBaseURL
	}
	return &elproofGateway{
		appID:   strings.TrimSpace(appID),
		secret:  strings.TrimSpace(secret),
		baseURL: base,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (g *elproofGateway) enabled() bool { return g.appID != "" && g.secret != "" }

// epErrors adalah bentuk `errors` ElProof — SATU OBJEK `{code}`, bukan array seperti envelope
// Elkasir sendiri (`.claude/rules/api-standard.md`). Jangan disamakan.
type epErrors struct {
	Code string `json:"code"`
}

type epTokenEnvelope struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Errors  epErrors `json:"errors"`
	Data    struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int64  `json:"expiresIn"`
	} `json:"data"`
}

// ensureToken mengembalikan access token yang masih valid, menukar ulang HANYA saat token yang
// di-cache sudah (hampir) kedaluwarsa — ElProof rate-limit endpoint token 10 req/menit per IP,
// jadi menukar ulang di setiap panggilan akan melanggar itu untuk trafik normal.
func (g *elproofGateway) ensureToken(ctx context.Context) (string, error) {
	g.tokenMu.Lock()
	defer g.tokenMu.Unlock()
	if g.cachedToken != "" && time.Now().Before(g.tokenExpiry) {
		return g.cachedToken, nil
	}

	body, err := json.Marshal(map[string]string{"appId": g.appID, "secret": g.secret})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/auth/app/token", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("elproof: token exchange gagal terkirim: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var env epTokenEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", fmt.Errorf("elproof: parse token response (HTTP %d): %w", resp.StatusCode, err)
	}
	if !env.Success {
		msg := env.Message
		if msg == "" {
			msg = string(raw)
		}
		return "", fmt.Errorf("elproof: token exchange ditolak (HTTP %d, code=%s): %s", resp.StatusCode, env.Errors.Code, msg)
	}
	g.cachedToken = env.Data.AccessToken
	// Margin aman 60 detik sebelum expiresIn benar-benar habis.
	g.tokenExpiry = time.Now().Add(time.Duration(env.Data.ExpiresIn)*time.Second - 60*time.Second)
	return g.cachedToken, nil
}

// epChargeEnvelope adalah bentuk response ElProof untuk PEMBUATAN charge maupun STATUS CHECK —
// keduanya SAMA PERSIS bentuknya (beda dari kontrak Elkasir sendiri, yang status-checknya jauh
// lebih tipis: {paid, rawStatus}).
type epChargeEnvelope struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Errors  epErrors `json:"errors"`
	Data    struct {
		OrderRef    string `json:"orderRef"`
		ProviderRef string `json:"providerRef"`
		Channel     string `json:"channel"`
		QRImageURL  string `json:"qrImageUrl"`
		PayCode     string `json:"payCode"`
		CheckoutURL string `json:"checkoutUrl"`
		Amount      int64  `json:"amount"`
		FeeAmount   int64  `json:"feeAmount"`
		ExpiresAt   string `json:"expiresAt"`
		Status      string `json:"status"` // unpaid | paid | expired | failed | refund
	} `json:"data"`
}

// createCharge hanya mengimplementasikan QRIS — subscription tidak pernah meminta kanal lain
// (YAGNI; virtual_account lewat ElProof belum dibutuhkan). ElProof tidak mengembalikan qrString
// mentah untuk QRIS (hanya qrImageUrl) — sudah diverifikasi frontend subscription (
// SubscriptionQrisPanel.tsx) merender <img> dari qrImageUrl bila tersedia, jadi ini aman.
//
// customerName/customerEmail/customerPhone SELALU disertakan meski didokumentasikan opsional —
// diverifikasi empiris (2026-07-19): tanpa field ini, ElProof membalas 500/internal generik;
// dengan field ini terisi, charge berhasil dibuat. ElProof membungkus Tripay, dan Tripay sendiri
// mewajibkan customer_name/customer_email di level API-nya (persis alasan tripay.go di modul ini
// juga meng-hardcode nilai default yang sama untuk selforder) — batasan ini bocor lewat ElProof
// walau dokumentasinya bilang opsional.
func (g *elproofGateway) createCharge(ctx context.Context, orderRef string, amount int64, channel paymentclient.Channel, _ paymentclient.ChannelOptions) (chargeResult, error) {
	if channel != paymentclient.ChannelQRIS && channel != "" {
		return chargeResult{}, fmt.Errorf("elproof: channel %q belum didukung (subscription hanya memakai QRIS)", channel)
	}
	token, err := g.ensureToken(ctx)
	if err != nil {
		return chargeResult{}, err
	}

	body, err := json.Marshal(map[string]any{
		"orderRef":      orderRef,
		"amount":        amount,
		"channel":       "QRIS",
		"customerName":  "Elkasir Billing",
		"customerEmail": "billing@elkasir.app",
		"customerPhone": "0800000000",
	})
	if err != nil {
		return chargeResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/external/payments/charges", bytes.NewReader(body))
	if err != nil {
		return chargeResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return chargeResult{}, fmt.Errorf("elproof: charge gagal terkirim: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var env epChargeEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return chargeResult{}, fmt.Errorf("elproof: parse charge response (HTTP %d): %w", resp.StatusCode, err)
	}
	if !env.Success {
		msg := env.Message
		if msg == "" {
			msg = string(raw)
		}
		return chargeResult{}, fmt.Errorf("elproof: charge ditolak (HTTP %d, code=%s): %s", resp.StatusCode, env.Errors.Code, msg)
	}
	return chargeResult{
		Ref:        env.Data.ProviderRef,
		QRImageURL: env.Data.QRImageURL,
	}, nil
}

// checkStatus dikunci ke ORDER REF (idempotency key milik pemanggil sendiri) — BUKAN providerRef
// seperti Tripay/Midtrans — karena endpoint ElProof adalah GET /external/payments/charges/
// {orderRef}/status. Untuk AppSubscribe, pemanggil (subscription's reconciler) mengirim invoice
// ID-nya sendiri di sini (persis nilai yang dikirim sebagai orderRef saat CreateChannelCharge).
func (g *elproofGateway) checkStatus(ctx context.Context, orderRef string) (paymentclient.ChargeStatus, error) {
	token, err := g.ensureToken(ctx)
	if err != nil {
		return paymentclient.ChargeStatus{}, err
	}

	u := g.baseURL + "/external/payments/charges/" + url.PathEscape(orderRef) + "/status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return paymentclient.ChargeStatus{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return paymentclient.ChargeStatus{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var env epChargeEnvelope
	if err := json.Unmarshal(raw, &env); err != nil || !env.Success {
		return paymentclient.ChargeStatus{}, fmt.Errorf("elproof: gagal memeriksa status (HTTP %d)", resp.StatusCode)
	}
	return paymentclient.ChargeStatus{
		Paid:      strings.EqualFold(strings.TrimSpace(env.Data.Status), "paid"),
		RawStatus: env.Data.Status,
	}, nil
}

// verifyWebhook: X-Webhook-Signature = HMAC-SHA256(raw_body, secret) — hex MENTAH, TANPA prefix
// "sha256=" (beda dari X-Elkasir-Signature milik Elkasir sendiri, yang punya prefix itu).
func (g *elproofGateway) verifyWebhook(header http.Header, body []byte) bool {
	if g.secret == "" {
		return false
	}
	got := strings.TrimSpace(header.Get("X-Webhook-Signature"))
	if got == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(g.secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(got))
}

type epWebhookPayload struct {
	OrderRef string `json:"orderRef"`
	Paid     bool   `json:"paid"`
}

// parseWebhook: payload ElProof TIDAK PUNYA eventId — orderRef dipakai ganda sebagai EventID
// (aman: webhook_events unique-key (provider, event_id) mengisolasi namespace "elproof" dari
// "tripay"/"midtrans" sepenuhnya).
func (g *elproofGateway) parseWebhook(body []byte) (paymentclient.WebhookEvent, error) {
	var p epWebhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return paymentclient.WebhookEvent{}, paymentclient.ErrInvalidPayload
	}
	return paymentclient.WebhookEvent{EventID: p.OrderRef, OrderRef: p.OrderRef, Paid: p.Paid}, nil
}
