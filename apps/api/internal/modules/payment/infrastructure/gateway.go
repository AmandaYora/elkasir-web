// Package infrastructure implements paymentclient.Client di atas SATU gateway QRIS aktif
// (Tripay / Midtrans) yang dipilih dari konfigurasi, atau mode simulasi bila tak ada yang
// aktif. Bagian yang TIDAK bergantung provider (pencatatan tabel payments + idempotensi
// webhook_events) hidup di client.go; detail tiap provider hidup di tripay.go / midtrans.go.
package infrastructure

import (
	"context"
	"net/http"
	"strings"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/platform/config"
)

// qrResult adalah hasil charge yang sudah dinormalisasi lintas provider.
//
//	Ref        = referensi transaksi milik provider (disimpan sebagai provider_ref)
//	QRString   = payload QRIS mentah (bila provider menyediakannya)
//	QRImageURL = URL gambar QR siap-tampil dari provider
type qrResult struct {
	Ref        string
	QRString   string
	QRImageURL string
}

// gateway adalah kontrak internal untuk satu penyedia QRIS. Semua perbedaan provider
// (endpoint, header auth, skema signature, format payload/callback) disembunyikan di sini
// sehingga apiClient tetap provider-agnostic.
type gateway interface {
	name() string // nilai kolom payments.provider / webhook_events.provider
	enabled() bool
	createCharge(ctx context.Context, orderRef string, amount int64) (qrResult, error)
	// quoteFee mengembalikan biaya gateway QRIS untuk `amount` (rupiah) yang akan ditagih.
	quoteFee(ctx context.Context, amount int64) (int64, error)
	verifyWebhook(header http.Header, body []byte) bool
	parseWebhook(body []byte) (paymentclient.WebhookEvent, error)
}

// selectGateway memilih gateway aktif dari konfigurasi. Mengembalikan nil (→ mode simulasi)
// bila provider tak dipilih atau kredensialnya belum lengkap.
func selectGateway(cfg config.Payment) gateway {
	switch cfg.ActiveProvider() {
	case "tripay":
		if g := newTripay(cfg.Tripay, cfg.CallbackURL); g.enabled() {
			return g
		}
	case "midtrans":
		if g := newMidtrans(cfg.Midtrans); g.enabled() {
			return g
		}
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" && t != ":" {
			return v
		}
	}
	return ""
}
