// Package paymentclient adalah KONTRAK modul payment (gateway pembayaran QRIS + tabel
// payments/webhook_events) untuk dikonsumsi modul selforder (≈ "Payment Client" pada diagram).
//
// Sengaja PROVIDER-AGNOSTIC: tidak ada asumsi gateway tertentu (Xendit/Midtrans/dll) di
// permukaan kontrak, sehingga interface ini stabil walau implementasi provider berganti.
// Detail provider (header token, skema signature, format payload) hidup di modul payment.
package paymentclient

import (
	"context"
	"errors"
	"net/http"
)

// Charge adalah hasil pembuatan tagihan QRIS. Simulated=true saat gateway tak dikonfigurasi
// (mode dev: pakai endpoint simulasi alih-alih gateway nyata).
//
// QRString = payload QRIS mentah (bila provider menyediakannya; dirender jadi QR di klien).
// QRImageURL = URL gambar QR siap-tampil dari provider (Midtrans QRIS hanya memberi ini).
// Klien menampilkan QRImageURL bila ada, jika tidak merender QR dari QRString.
type Charge struct {
	QRString    string
	QRImageURL  string
	ProviderRef string
	Simulated   bool
}

// WebhookEvent adalah hasil parse callback gateway (dinormalisasi).
type WebhookEvent struct {
	EventID  string // identitas unik event (idempotensi)
	OrderRef string // = id self-order
	Paid     bool
}

// Client adalah kontrak yang dipublikasikan modul payment.
type Client interface {
	Enabled() bool
	// QuoteFee mengembalikan biaya gateway QRIS untuk `amount` (rupiah). 0 bila gateway
	// nonaktif (mode simulasi). Dipakai untuk menampilkan/menagih biaya ke pelanggan.
	QuoteFee(ctx context.Context, amount int64) (int64, error)
	CreateCharge(ctx context.Context, storeID, orderID string, amount int64) (Charge, error) // create()
	// VerifyWebhook memvalidasi keaslian callback. Header & body diserahkan apa adanya;
	// PROVIDER yang menentukan skema verifikasi (header token, signature HMAC atas body, dll)
	// — supaya kontrak tak perlu berubah saat ganti gateway.
	VerifyWebhook(header http.Header, body []byte) bool
	ParseWebhook(body []byte) (WebhookEvent, error)
	WebhookSeen(ctx context.Context, eventID string) (bool, error)
	MarkWebhookSeen(ctx context.Context, eventID string) error
}

// ErrInvalidPayload: body webhook tidak dapat di-parse.
var ErrInvalidPayload = errors.New("payload webhook tidak valid")
