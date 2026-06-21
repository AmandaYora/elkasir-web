// Gateway Tripay (Closed Payment — QRIS, sandbox/production). API Key untuk Bearer auth;
// Private Key untuk signature charge (HMAC-SHA256(merchantCode+merchantRef+amount)) dan
// verifikasi signature callback (HMAC-SHA256 atas raw body, header X-Callback-Signature).
package infrastructure

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/platform/config"
)

type tripayGateway struct {
	apiKey       string
	privateKey   string
	merchantCode string
	method       string // kode channel QRIS (mis. "QRIS")
	baseURL      string
	callbackURL  string
	http         *http.Client
}

func newTripay(cfg config.Tripay, callbackURL string) *tripayGateway {
	method := strings.TrimSpace(cfg.Method)
	if method == "" {
		method = "QRIS"
	}
	return &tripayGateway{
		apiKey:       strings.TrimSpace(cfg.APIKey),
		privateKey:   strings.TrimSpace(cfg.PrivateKey),
		merchantCode: strings.TrimSpace(cfg.MerchantCode),
		method:       method,
		baseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		callbackURL:  callbackURL,
		http:         &http.Client{Timeout: 15 * time.Second},
	}
}

func (g *tripayGateway) name() string { return "tripay" }
func (g *tripayGateway) enabled() bool {
	return g.apiKey != "" && g.privateKey != "" && g.merchantCode != ""
}

func (g *tripayGateway) hmac(msg string) string {
	mac := hmac.New(sha256.New, []byte(g.privateKey))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// ── Charge (transaction/create) ──────────────────────────────
type tpEnvelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    tpChargeData    `json:"data"`
	Errors  json.RawMessage `json:"errors,omitempty"`
}

type tpChargeData struct {
	Reference   string `json:"reference"`
	MerchantRef string `json:"merchant_ref"`
	QRString    string `json:"qr_string"`
	QRURL       string `json:"qr_url"`
	CheckoutURL string `json:"checkout_url"`
	Status      string `json:"status"`
}

func (g *tripayGateway) createCharge(ctx context.Context, orderRef string, amount int64) (qrResult, error) {
	// signature = HMAC-SHA256(merchant_code + merchant_ref + amount, private_key)
	sig := g.hmac(g.merchantCode + orderRef + strconv.FormatInt(amount, 10))
	amountStr := strconv.FormatInt(amount, 10)

	// Tripay transaction/create mengharapkan body x-www-form-urlencoded (order_items sebagai
	// array bracket-notation), bukan JSON — lihat dokumentasi resmi.
	form := url.Values{}
	form.Set("method", g.method)
	form.Set("merchant_ref", orderRef)
	form.Set("amount", amountStr)
	form.Set("customer_name", "Pelanggan Elkasir")
	form.Set("customer_email", "noreply@elkasir.app")
	form.Set("order_items[0][sku]", orderRef)
	form.Set("order_items[0][name]", "Pesanan Elkasir")
	form.Set("order_items[0][price]", amountStr)
	form.Set("order_items[0][quantity]", "1")
	if g.callbackURL != "" {
		form.Set("callback_url", g.callbackURL)
	}
	form.Set("signature", sig)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/transaction/create", strings.NewReader(form.Encode()))
	if err != nil {
		return qrResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return qrResult{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var env tpEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return qrResult{}, fmt.Errorf("tripay: parse response (HTTP %d): %w", resp.StatusCode, err)
	}
	if !env.Success {
		msg := env.Message
		if msg == "" {
			msg = string(raw)
		}
		return qrResult{}, fmt.Errorf("tripay: charge ditolak (HTTP %d): %s", resp.StatusCode, msg)
	}
	return qrResult{
		Ref:        env.Data.Reference,
		QRString:   env.Data.QRString,
		QRImageURL: firstNonEmpty(env.Data.QRURL, env.Data.CheckoutURL),
	}, nil
}

// ── Fee calculator (live) ────────────────────────────────────
type tpFeeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    []struct {
		Code     string `json:"code"`
		TotalFee struct {
			Merchant int64 `json:"merchant"`
			Customer int64 `json:"customer"`
		} `json:"total_fee"`
	} `json:"data"`
}

// tarif QRIS standar Tripay (Rp750 + 0,7%) — dipakai sebagai fallback bila kalkulator live
// tak terjangkau agar checkout tidak terblokir. Untuk QRIS, total_fee.merchant = nilai ini.
const (
	tripayQRISFeeFlat    int64 = 750
	tripayQRISFeePerMille int64 = 7 // 0,7%
)

func tripayQRISFeeFallback(amount int64) int64 {
	if amount <= 0 {
		return 0
	}
	// Bagian persen dibulatkan KE ATAS (ceiling) agar fallback tak pernah under-quote fee
	// gateway — merchant tidak pernah kurang bayar (arah aman saat kalkulator live down).
	percentFee := (amount*tripayQRISFeePerMille + 999) / 1000
	return tripayQRISFeeFlat + percentFee
}

// quoteFee memanggil GET /merchant/fee-calculator?code=&amount= dan mengembalikan
// total_fee.merchant untuk channel QRIS aktif (biaya yang akan dibebankan ke pelanggan).
// Bila panggilan live gagal, jatuh ke rumus standar (tidak pernah mengembalikan error).
func (g *tripayGateway) quoteFee(ctx context.Context, amount int64) (int64, error) {
	u := fmt.Sprintf("%s/merchant/fee-calculator?code=%s&amount=%d",
		g.baseURL, url.QueryEscape(g.method), amount)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return tripayQRISFeeFallback(amount), nil
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		slog.Warn("tripay fee-calculator gagal; pakai fallback", "err", err)
		return tripayQRISFeeFallback(amount), nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var fr tpFeeResponse
	if err := json.Unmarshal(raw, &fr); err != nil || !fr.Success || len(fr.Data) == 0 {
		slog.Warn("tripay fee-calculator tak valid; pakai fallback", "status", resp.StatusCode, "msg", fr.Message)
		return tripayQRISFeeFallback(amount), nil
	}
	for _, d := range fr.Data {
		if strings.EqualFold(d.Code, g.method) {
			return d.TotalFee.Merchant, nil
		}
	}
	return fr.Data[0].TotalFee.Merchant, nil
}

// ── Callback ─────────────────────────────────────────────────
type tpCallback struct {
	Reference   string `json:"reference"`
	MerchantRef string `json:"merchant_ref"`
	Status      string `json:"status"` // PAID | UNPAID | EXPIRED | FAILED | REFUND
}

// verifyWebhook: X-Callback-Signature = HMAC-SHA256(raw_body, private_key). Optional sanity:
// X-Callback-Event = "payment_status".
func (g *tripayGateway) verifyWebhook(header http.Header, body []byte) bool {
	if g.privateKey == "" {
		return false
	}
	got := strings.TrimSpace(header.Get("X-Callback-Signature"))
	if got == "" {
		return false
	}
	if ev := header.Get("X-Callback-Event"); ev != "" && !strings.EqualFold(ev, "payment_status") {
		return false
	}
	return hmac.Equal([]byte(g.hmac(string(body))), []byte(got))
}

func (g *tripayGateway) parseWebhook(body []byte) (paymentclient.WebhookEvent, error) {
	var p tpCallback
	if err := json.Unmarshal(body, &p); err != nil {
		return paymentclient.WebhookEvent{}, paymentclient.ErrInvalidPayload
	}
	return paymentclient.WebhookEvent{
		EventID:  firstNonEmpty(p.Reference+":"+p.Status, p.MerchantRef+":"+p.Status),
		OrderRef: p.MerchantRef,
		Paid:     strings.EqualFold(strings.TrimSpace(p.Status), "PAID"),
	}, nil
}
