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
	"time"
)

// AppID mengidentifikasi konsumen gateway ini — dua nilai bawaan sistem (di-seed migrasi
// 000019_payment_gateway_registry) untuk dua konsumen internal yang sudah ada. Konsumen baru
// (internal atau eksternal, §9.7) didaftarkan lewat registry (ListApps/CreateApp) tanpa perlu
// ubah kontrak ini.
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

// ChannelInfo menjelaskan satu kanal yang saat ini aktif/tersedia di akun gateway — dilaporkan
// live dari provider (ListChannels), bukan daftar statis dalam kode (§9.1.8).
type ChannelInfo struct {
	Channel Channel `json:"channel"`
	Code    string  `json:"code"`   // kode provider (mis. "QRIS", "BCAVA")
	Name    string  `json:"name"`   // nama tampilan (mis. "BCA Virtual Account")
	Active  bool    `json:"active"`
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
	// ResolveApp resolves a payment_clients row ID (an ActorApp principal's SubjectID) to its
	// AppInfo — used by the external charge-creation route to know which app_id is calling.
	ResolveApp(ctx context.Context, rowID string) (AppInfo, error)
	// ResolveCharge translates a caller-supplied orderRef into the owning app_id and the
	// gateway's own providerRef (which CheckStatus actually needs) — used by the external
	// status-check route, which also verifies the resolved app_id matches the caller before
	// proceeding (an app must never be able to probe another app's charge by guessing orderRef).
	ResolveCharge(ctx context.Context, orderRef string) (appID, providerRef string, err error)
}

// AppInfo adalah satu baris registry (§9.1.3) — dipakai UI Konsol Platform "Aplikasi Terdaftar".
type AppInfo struct {
	ID          string    `json:"id"`
	AppID       string    `json:"appId"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"` // "internal" | "external"
	CallbackURL string    `json:"callbackUrl"`
	Status      string    `json:"status"` // "active" | "inactive"
	CreatedAt   time.Time `json:"createdAt"`
}

// CreateAppInput mendaftarkan app BARU — selalu kind="external" (dua app internal sudah
// di-seed migrasi, tidak dibuat lewat endpoint ini, §9.1.3).
type CreateAppInput struct {
	Name        string `json:"name"`
	CallbackURL string `json:"callbackUrl"`
}

// CreateAppResult menyertakan Secret HANYA pada respons pembuatan — tidak pernah lagi
// ditampilkan setelahnya (§9.1.3/§9.3 PF1).
type CreateAppResult struct {
	AppInfo
	Secret string `json:"secret"`
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
}

// Client adalah kontrak yang dipublikasikan modul payment.
type Client interface {
	Enabled() bool
	// QuoteFee mengembalikan biaya gateway QRIS untuk `amount` (rupiah). 0 bila gateway
	// nonaktif (mode simulasi). Dipakai untuk menampilkan/menagih biaya ke pelanggan.
	QuoteFee(ctx context.Context, amount int64) (int64, error)
	// CreateCharge adalah wrapper QRIS-khusus di atas CreateChannelCharge (§9.1.8) — tetap ada,
	// perilaku tak berubah, supaya selforder/subscription tak perlu ubah cara memanggilnya.
	CreateCharge(ctx context.Context, appID, storeID, orderID string, amount int64) (Charge, error)
	CreateChannelCharge(ctx context.Context, appID, storeID, orderID string, amount int64, channel Channel, opts ChannelOptions) (Charge, error)
	// ListChannels melaporkan kanal yang AKTIF di akun gateway saat ini — live dari provider,
	// bukan daftar statis (§9.1.8).
	ListChannels(ctx context.Context) ([]ChannelInfo, error)
	// CheckStatus adalah pull-based status check, independen dari webhook (§9.1.8).
	CheckStatus(ctx context.Context, providerRef string) (ChargeStatus, error)
	// VerifyWebhook memvalidasi keaslian callback. Header & body diserahkan apa adanya;
	// PROVIDER yang menentukan skema verifikasi (header token, signature HMAC atas body, dll)
	// — supaya kontrak tak perlu berubah saat ganti gateway.
	VerifyWebhook(header http.Header, body []byte) bool
	ParseWebhook(body []byte) (WebhookEvent, error)
	WebhookSeen(ctx context.Context, eventID string) (bool, error)
	MarkWebhookSeen(ctx context.Context, eventID string) error

	// ── Config gateway (§9.1.2/§9.1.6) — dikonsumsi modul `platform`, yang mengekspos
	// route superadmin-nya sendiri di /platform/payment-config (§9.1.10). ──────────────────
	GetConfig(ctx context.Context) (GatewayConfig, error)
	UpdateConfig(ctx context.Context, in UpdateGatewayConfigInput) (GatewayConfig, error)

	// ── Registry app (§9.1.3) — dikonsumsi modul `platform`, yang mengekspos route
	// superadmin-nya sendiri di /platform/payment-clients (§9.1.10). ─────────────────────────
	ListApps(ctx context.Context) ([]AppInfo, error)
	CreateApp(ctx context.Context, in CreateAppInput) (CreateAppResult, error)
	ResetAppSecret(ctx context.Context, id string) (string, error)
	SetAppStatus(ctx context.Context, id, status string) error
}

// ErrInvalidPayload: body webhook tidak dapat di-parse.
var ErrInvalidPayload = errors.New("payload webhook tidak valid")
