// Package paymentclient adalah KONTRAK modul payment (gateway pembayaran QRIS/VA + registry app
// + config gateway) untuk dikonsumsi modul selforder/subscription/platform (≈ "Payment Client"
// pada diagram).
//
// Sengaja PROVIDER-AGNOSTIC: tidak ada asumsi gateway tertentu (Tripay/Midtrans/dll) di
// permukaan kontrak, sehingga interface ini stabil walau implementasi provider berganti.
// Detail provider (header token, skema signature, format payload) hidup di modul payment.
//
// PLAN.md §9 (Part 2): payment tetap SATU dompet (satu kredensial gateway aktif, §9.1.1) —
// menambah "app" (AppID) tidak pernah berarti kredensial baru, hanya identitas atribusi + tujuan
// dispatch webhook. Dua konsumen internal yang sudah ada dipromosikan jadi app terdaftar formal,
// menggantikan trik prefix "sub_" yang sebelumnya dipakai `subscription/domain`.
package paymentclient

import (
	"context"
	"errors"
	"net/http"
)

// AppID mengidentifikasi konsumen gateway ini — dua nilai bawaan sistem (di-seed migrasi
// 000019_payment_gateway_registry) untuk dua konsumen internal yang sudah ada. AppSubscribe kini
// juga menjadi kunci untuk memilih gateway ElProof di CreateChannelCharge/CheckStatus (§11) — ini
// tetap identitas dispatch INTERNAL Elkasir, bukan identitas Elkasir di sisi ElProof sendiri
// (yang itu appId "Elkasir-Billing", didaftarkan terpisah di Platform Console ElProof).
const (
	AppSelfOrder = "ELKASIR-SELFORDER"
	AppSubscribe = "ELKASIR-SUBSCRIBE"
)

// Channel adalah kanal pembayaran yang didukung CreateChannelCharge. Enum SENGAJA konservatif
// (§9.1.8, diputuskan 2026-07-12) — retail/e-wallet belum diaktifkan, tambah nanti bila memang
// dibutuhkan, bukan sekarang.
type Channel string

const (
	ChannelQRIS Channel = "qris"
	ChannelVA   Channel = "virtual_account"
)

// ChannelOptions membawa parameter tambahan spesifik kanal. BankCode HANYA dipakai untuk
// ChannelVA, dan HARUS berupa kode yang benar-benar aktif di akun gateway (lihat ListChannels) —
// tidak ada daftar bank yang di-hardcode di sini (§9.1.8).
type ChannelOptions struct {
	BankCode string // wajib untuk ChannelVA (mis. "BCAVA", "MANDIRIVA" — kode Tripay)
}

// Charge adalah hasil pembuatan tagihan. Simulated=true saat gateway tak dikonfigurasi
// (mode dev: pakai endpoint simulasi alih-alih gateway nyata).
//
// QRString/QRImageURL berlaku untuk ChannelQRIS. VANumber/VABankCode berlaku untuk ChannelVA
// (nomor rekening virtual yang pelanggan transfer ke sana, bukan gambar QR).
//
// Provider = nama provider AKTIF yang membuat charge ini ("tripay" | "midtrans", atau default
// "midtrans" dalam mode simulasi). Modul payment TIDAK mencatat baris ledger bisnis apa pun
// (lihat paket infrastructure) — pemanggil (selforder, subscription, dst.) memakai field ini
// untuk mencatat baris ledger MILIKNYA SENDIRI, sehingga data tiap domain bisnis tetap terpisah.
type Charge struct {
	Channel     Channel `json:"channel"`
	QRString    string  `json:"qrString"`
	QRImageURL  string  `json:"qrImageUrl"`
	VANumber    string  `json:"vaNumber"`
	VABankCode  string  `json:"vaBankCode"`
	ProviderRef string  `json:"providerRef"`
	Provider    string  `json:"provider"`
	Simulated   bool    `json:"simulated"`
}

// ChargeStatus adalah hasil pull-based status check (CheckStatus) — independen dari jalur
// webhook push, untuk kasus webhook terlambat/hilang (§9.1.8).
type ChargeStatus struct {
	Paid      bool   `json:"paid"`
	RawStatus string `json:"rawStatus"` // status mentah dari provider, untuk logging/debug
}

// WebhookEvent adalah hasil parse callback gateway (dinormalisasi).
type WebhookEvent struct {
	EventID  string `json:"eventId"`  // identitas unik event (idempotensi)
	OrderRef string `json:"orderRef"` // = order ref yang dikirim saat CreateCharge (tanpa prefix apa pun, §9.1.4)
	Paid     bool   `json:"paid"`
}

// WebhookConsumer diimplementasikan oleh setiap modul konsumen (internal, kind='internal') yang
// bisa menerapkan event gateway yang SUDAH diverifikasi/di-parse/dicek-idempotensi ke data
// domainnya sendiri. Didaftarkan lewat payment.Module.RegisterConsumer di composition root
// (app.go) — bukan bagian dari Client di bawah, karena ini relasi "siapa menerima dispatch",
// bukan "siapa memanggil gateway" (§9.1.5).
type WebhookConsumer interface {
	ApplyWebhookEvent(ctx context.Context, ev WebhookEvent) error
}

// Dispatcher is the composition-root/presentation-only counterpart to Client — registering
// consumers, dispatching an already-verified webhook event to whichever one owns it, and (Part
// 3, §10.2 EB2) resolving app/charge identity for payment's own external routes. Kept separate
// from Client so business modules (selforder, subscription) never see it; only app.go (wiring)
// and payment/presentation (the webhook handler + external routes) type-assert a Client down to
// this.
type Dispatcher interface {
	RegisterConsumer(appID string, consumer WebhookConsumer)
	Dispatch(ctx context.Context, ev WebhookEvent) error
}

// GatewayConfig adalah representasi config gateway yang AMAN ditampilkan ke UI — field secret
// selalu termask (mis. "••••1234"), tidak pernah nilai asli (§9.1.2).
type GatewayConfig struct {
	Provider                string `json:"provider"`
	Sandbox                 bool   `json:"sandbox"`
	TripayAPIKeyMasked      string `json:"tripayApiKeyMasked"`
	TripayPrivateKeyMasked  string `json:"tripayPrivateKeyMasked"`
	TripayMerchantCode      string `json:"tripayMerchantCode"` // bukan secret — aman ditampilkan apa adanya
	TripayMethod            string `json:"tripayMethod"`
	MidtransServerKeyMasked string `json:"midtransServerKeyMasked"`
	// ElProof adalah dompet TERPISAH yang HANYA dipakai untuk billing subscription (appID
	// AppSubscribe) — selalu aktif berdampingan dengan Provider di atas (dipakai selforder),
	// bukan dipilih lewat switch yang sama. Lihat PLAN.md §11.
	ElProofAppID        string `json:"elproofAppId"`
	ElProofSecretMasked string `json:"elproofSecretMasked"`
	ElProofBaseURL      string `json:"elproofBaseUrl"`
}

// UpdateGatewayConfigInput — field secret bertipe pointer: nil = "jangan ubah" (biarkan nilai
// terenkripsi yang sudah ada), string kosong pun HARUS eksplisit (jangan dikirim pointer ke ""
// kecuali memang bermaksud mengosongkannya). Ini yang membuat form write-only di UI aman —
// submit tanpa mengetik ulang secret tidak menghapusnya (§9.1.2/§9.3 PF0).
type UpdateGatewayConfigInput struct {
	Provider           string  `json:"provider"`
	Sandbox            bool    `json:"sandbox"`
	TripayAPIKey       *string `json:"tripayApiKey"`
	TripayPrivateKey   *string `json:"tripayPrivateKey"`
	TripayMerchantCode *string `json:"tripayMerchantCode"`
	TripayMethod       string  `json:"tripayMethod"`
	MidtransServerKey  *string `json:"midtransServerKey"`
	ElProofAppID       *string `json:"elproofAppId"`
	ElProofSecret      *string `json:"elproofSecret"`
	ElProofBaseURL     string  `json:"elproofBaseUrl"`
}

// Client adalah kontrak yang dipublikasikan modul payment.
type Client interface {
	Enabled() bool
	// ActiveProviderName melaporkan label provider Tripay/Midtrans aktif saat ini (default
	// "midtrans" dalam mode simulasi) — dipakai presentation/handler.go untuk mengunci
	// idempotensi webhook Elkasir sendiri (WebhookSeen/MarkWebhookSeen) ke provider yang benar.
	// ElProof tidak pernah lewat sini — ia punya namespace "elproof" literal sendiri (§11).
	ActiveProviderName() string
	// QuoteFee mengembalikan biaya gateway QRIS untuk `amount` (rupiah). 0 bila gateway
	// nonaktif (mode simulasi). Dipakai untuk menampilkan/menagih biaya ke pelanggan.
	QuoteFee(ctx context.Context, amount int64) (int64, error)
	// CreateCharge adalah wrapper QRIS-khusus di atas CreateChannelCharge (§9.1.8) — tetap ada,
	// perilaku tak berubah, supaya selforder/subscription tak perlu ubah cara memanggilnya.
	CreateCharge(ctx context.Context, appID, storeID, orderID string, amount int64) (Charge, error)
	// CreateChannelCharge membuat tagihan lewat gateway yang sesuai untuk appID ini: appID
	// AppSubscribe selalu lewat ElProof (dompet TERPISAH khusus subscription billing); appID
	// lain (mis. AppSelfOrder) tetap lewat SATU wallet Tripay/Midtrans aktif seperti sebelumnya.
	// Pembukaan eksplisit keputusan LOCKED §9.1.1 ("satu kredensial gateway aktif secara
	// global") — lihat PLAN.md §11 untuk alasannya.
	CreateChannelCharge(ctx context.Context, appID, storeID, orderID string, amount int64, channel Channel, opts ChannelOptions) (Charge, error)
	// CheckStatus adalah pull-based status check, independen dari webhook (§9.1.8). appID
	// menentukan gateway mana yang diperiksa (lihat CreateChannelCharge) — providerRef Tripay/
	// Midtrans dan providerRef ElProof hidup di namespace yang sepenuhnya terpisah.
	CheckStatus(ctx context.Context, appID, providerRef string) (ChargeStatus, error)
	// VerifyWebhook/ParseWebhook memvalidasi & mem-parsing callback dari wallet Tripay/Midtrans
	// Elkasir sendiri. PROVIDER yang menentukan skema verifikasi — supaya kontrak tak perlu
	// berubah saat ganti gateway.
	VerifyWebhook(header http.Header, body []byte) bool
	ParseWebhook(body []byte) (WebhookEvent, error)
	// VerifyElProofWebhook/ParseElProofWebhook adalah pasangannya untuk relay ElProof — skema
	// signature (X-Webhook-Signature, TANPA prefix "sha256=") dan bentuk payload (TANPA eventId)
	// berbeda total dari Tripay/Midtrans, sehingga tidak berbagi implementasi dengan pasangan di
	// atas. Lihat infrastructure/elproof.go.
	VerifyElProofWebhook(header http.Header, body []byte) bool
	ParseElProofWebhook(body []byte) (WebhookEvent, error)
	// WebhookSeen/MarkWebhookSeen mengambil `provider` eksplisit (bukan lagi diam-diam dari
	// gateway aktif) — supaya idempotensi ElProof ("elproof") tidak pernah tercampur dengan
	// idempotensi Tripay/Midtrans, walau disimpan di tabel webhook_events yang sama (unique key
	// (provider, event_id) sudah mengisolasi keduanya).
	WebhookSeen(ctx context.Context, provider, eventID string) (bool, error)
	MarkWebhookSeen(ctx context.Context, provider, eventID string) error

	// ── Config gateway (§9.1.2/§9.1.6) — dikonsumsi modul `platform`, yang mengekspos
	// route superadmin-nya sendiri di /platform/payment-config (§9.1.10). ──────────────────
	GetConfig(ctx context.Context) (GatewayConfig, error)
	UpdateConfig(ctx context.Context, in UpdateGatewayConfigInput) (GatewayConfig, error)
}

// ErrInvalidPayload: body webhook tidak dapat di-parse.
var ErrInvalidPayload = errors.New("payload webhook tidak valid")
