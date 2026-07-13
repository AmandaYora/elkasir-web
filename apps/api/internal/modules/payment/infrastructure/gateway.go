// Package infrastructure implements paymentclient.Client di atas SATU gateway QRIS/VA aktif
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

// chargeResult adalah hasil charge yang sudah dinormalisasi lintas provider/kanal.
//
//	Ref        = referensi transaksi milik provider (disimpan sebagai provider_ref)
//	QRString   = payload QRIS mentah (channel QRIS, bila provider menyediakannya)
//	QRImageURL = URL gambar QR siap-tampil (channel QRIS)
//	VANumber   = nomor rekening virtual (channel VA)
//	VABankCode = kode bank VA yang dipakai (channel VA)
type chargeResult struct {
	Ref        string
	QRString   string
	QRImageURL string
	VANumber   string
	VABankCode string
}

// gateway adalah kontrak internal untuk satu penyedia pembayaran. Semua perbedaan provider
// (endpoint, header auth, skema signature, format payload/callback) disembunyikan di sini
// sehingga apiClient tetap provider-agnostic.
type gateway interface {
	name() string
	enabled() bool
	createCharge(ctx context.Context, orderRef string, amount int64, channel paymentclient.Channel, opts paymentclient.ChannelOptions) (chargeResult, error)
	// quoteFee mengembalikan biaya gateway QRIS untuk `amount` (rupiah) yang akan ditagih.
	quoteFee(ctx context.Context, amount int64) (int64, error)
	// listChannels melaporkan kanal yang aktif di akun gateway saat ini (§9.1.8) — live dari
	// provider, bukan daftar statis dalam kode.
	listChannels(ctx context.Context) ([]paymentclient.ChannelInfo, error)
	// checkStatus adalah pull-based status check, independen dari webhook (§9.1.8).
	checkStatus(ctx context.Context, providerRef string) (paymentclient.ChargeStatus, error)
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
