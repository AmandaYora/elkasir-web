// Gateway Midtrans (Core API — QRIS, sandbox/production). Server Key dipakai untuk Basic Auth
// charge DAN verifikasi signature webhook (SHA512(order_id+status_code+gross_amount+ServerKey)).
package infrastructure

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/platform/config"
)

// acquirer default QRIS. "gopay" menghasilkan QRIS universal (dapat dipindai semua aplikasi
// bank/e-wallet); settlement dirutekan via GoPay. Alternatif: "airpay shopee".
const qrisAcquirer = "gopay"

type midtransGateway struct {
	serverKey string
	baseURL   string
	http      *http.Client
}

func newMidtrans(cfg config.Midtrans) *midtransGateway {
	return &midtransGateway{
		serverKey: strings.TrimSpace(cfg.ServerKey),
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

func (g *midtransGateway) name() string  { return "midtrans" }
func (g *midtransGateway) enabled() bool { return g.serverKey != "" }

// ── Charge ───────────────────────────────────────────────────
type mtChargeRequest struct {
	PaymentType        string             `json:"payment_type"`
	TransactionDetails mtTransactionDetls `json:"transaction_details"`
	QRIS               mtQRISOptions      `json:"qris"`
}

type mtTransactionDetls struct {
	OrderID     string `json:"order_id"`
	GrossAmount int64  `json:"gross_amount"`
}

type mtQRISOptions struct {
	Acquirer string `json:"acquirer"`
}

type mtChargeAction struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type mtChargeResponse struct {
	StatusCode    string           `json:"status_code"`
	StatusMessage string           `json:"status_message"`
	TransactionID string           `json:"transaction_id"`
	OrderID       string           `json:"order_id"`
	QRString      string           `json:"qr_string"`
	Actions       []mtChargeAction `json:"actions"`
}

// createCharge saat ini hanya mengimplementasikan ChannelQRIS (perilaku tak berubah dari
// sebelum Part 2). ChannelVA sengaja BELUM diimplementasikan untuk Midtrans — Tripay adalah
// provider yang aktif hari ini (PAYMENT_PROVIDER=tripay); mengerjakan VA-nya Midtrans sebelum
// benar-benar dipakai di produksi hanya kerja spekulatif (lihat PLAN.md §9.2 PB1's note).
func (g *midtransGateway) createCharge(ctx context.Context, orderRef string, amount int64, channel paymentclient.Channel, _ paymentclient.ChannelOptions) (chargeResult, error) {
	if channel != paymentclient.ChannelQRIS && channel != "" {
		return chargeResult{}, fmt.Errorf("midtrans: channel %q belum didukung", channel)
	}
	body, _ := json.Marshal(mtChargeRequest{
		PaymentType:        "qris",
		TransactionDetails: mtTransactionDetls{OrderID: orderRef, GrossAmount: amount},
		QRIS:               mtQRISOptions{Acquirer: qrisAcquirer},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/v2/charge", bytes.NewReader(body))
	if err != nil {
		return chargeResult{}, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(g.serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return chargeResult{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var cr mtChargeResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return chargeResult{}, fmt.Errorf("midtrans: parse response (HTTP %d): %w", resp.StatusCode, err)
	}
	// Midtrans membalas status_code "201" saat QRIS dibuat; selain 2xx = gagal.
	if !strings.HasPrefix(cr.StatusCode, "2") {
		msg := cr.StatusMessage
		if msg == "" {
			msg = string(raw)
		}
		return chargeResult{}, fmt.Errorf("midtrans: charge ditolak (status_code %s): %s", cr.StatusCode, msg)
	}
	return chargeResult{
		Ref:        cr.TransactionID,
		QRString:   cr.QRString,
		QRImageURL: mtPickQRImageURL(cr.Actions),
	}, nil
}

// listChannels: hanya QRIS diketahui aktif untuk Midtrans dalam implementasi ini (lihat
// createCharge di atas) — bukan daftar lengkap kanal yang Midtrans sendiri dukung.
func (g *midtransGateway) listChannels(_ context.Context) ([]paymentclient.ChannelInfo, error) {
	if !g.enabled() {
		return nil, nil
	}
	return []paymentclient.ChannelInfo{{Channel: paymentclient.ChannelQRIS, Code: "qris", Name: "QRIS", Active: true}}, nil
}

// checkStatus memanggil GET /v2/{order_id}/status — pull-based, independen webhook.
func (g *midtransGateway) checkStatus(ctx context.Context, providerRef string) (paymentclient.ChargeStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+"/v2/"+url.PathEscape(providerRef)+"/status", nil)
	if err != nil {
		return paymentclient.ChargeStatus{}, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(g.serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return paymentclient.ChargeStatus{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var sr struct {
		TransactionStatus string `json:"transaction_status"`
		FraudStatus       string `json:"fraud_status"`
	}
	if err := json.Unmarshal(raw, &sr); err != nil {
		return paymentclient.ChargeStatus{}, fmt.Errorf("midtrans: gagal mengambil status transaksi (HTTP %d)", resp.StatusCode)
	}
	return paymentclient.ChargeStatus{
		Paid:      mtIsPaid(sr.TransactionStatus) && mtFraudAccepted(sr.FraudStatus),
		RawStatus: sr.TransactionStatus,
	}, nil
}

// mtPickQRImageURL memilih URL gambar QR dari actions: utamakan "generate-qr-code",
// lalu "generate-qr-code-v2", terakhir action pertama yang punya URL.
func mtPickQRImageURL(actions []mtChargeAction) string {
	var v2, first string
	for _, a := range actions {
		switch a.Name {
		case "generate-qr-code":
			return a.URL
		case "generate-qr-code-v2":
			v2 = a.URL
		}
		if first == "" && a.URL != "" {
			first = a.URL
		}
	}
	if v2 != "" {
		return v2
	}
	return first
}

// quoteFee mengembalikan estimasi biaya QRIS Midtrans. Midtrans tidak menyediakan endpoint
// kalkulator publik seperti Tripay, jadi dipakai tarif MDR QRIS standar 0,7% (tanpa flat).
func (g *midtransGateway) quoteFee(_ context.Context, amount int64) (int64, error) {
	if amount <= 0 {
		return 0, nil
	}
	return amount * 7 / 1000, nil
}

// ── Webhook (HTTP notification) ──────────────────────────────
// Field gross_amount & status_code bertipe STRING dan dipakai apa adanya untuk signature.
type mtWebhookPayload struct {
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
}

// verifyWebhook: signature_key = SHA512(order_id+status_code+gross_amount+ServerKey).
func (g *midtransGateway) verifyWebhook(_ http.Header, body []byte) bool {
	var p mtWebhookPayload
	if err := json.Unmarshal(body, &p); err != nil || g.serverKey == "" || p.SignatureKey == "" {
		return false
	}
	sum := sha512.Sum512([]byte(p.OrderID + p.StatusCode + p.GrossAmount + g.serverKey))
	// Bandingkan konstan-waktu (anti timing attack). gross_amount sudah ikut ditandatangani,
	// jadi nominal tak bisa dipalsukan tanpa menggagalkan signature ini.
	expected := hex.EncodeToString(sum[:])
	got := strings.ToLower(strings.TrimSpace(p.SignatureKey))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(got)) == 1
}

func (g *midtransGateway) parseWebhook(body []byte) (paymentclient.WebhookEvent, error) {
	var p mtWebhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return paymentclient.WebhookEvent{}, paymentclient.ErrInvalidPayload
	}
	// EventID unik per (transaksi, status) agar notifikasi "pending" lalu "settlement" untuk
	// transaksi yang sama TIDAK saling menumpuk pada idempotensi (settlement tetap diproses).
	return paymentclient.WebhookEvent{
		EventID:  firstNonEmpty(p.TransactionID+":"+p.TransactionStatus, p.OrderID+":"+p.TransactionStatus),
		OrderRef: p.OrderID,
		Paid:     mtIsPaid(p.TransactionStatus) && mtFraudAccepted(p.FraudStatus),
	}, nil
}

// mtIsPaid: status Midtrans yang berarti dana diterima. QRIS = "settlement"; "capture"
// disertakan untuk kelengkapan (kartu/akuisisi yang langsung capture).
func mtIsPaid(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "settlement", "capture":
		return true
	default:
		return false
	}
}

// mtFraudAccepted: untuk QRIS fraud_status biasanya kosong; bila ada, hanya "accept" lolos.
func mtFraudAccepted(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	return t == "" || t == "accept"
}
