// apiClient mengimplementasikan paymentclient.Client (dan paymentclient.Dispatcher, hanya
// dipakai composition root + presentation) di atas SATU gateway Tripay/Midtrans aktif (lihat
// gateway.go) PLUS satu gateway ElProof yang SELALU berdampingan, khusus billing subscription
// (elproof.go, PLAN.md §11) + konfigurasi gateway yang di-DB-kan (§9.1.2). Modul ini SENGAJA
// tidak mencatat baris ledger bisnis apa pun (mis. tabel self-order/subscription) — itu tanggung
// jawab masing-masing pemanggil (lihat paymentclient.Charge.Provider). State yang dipegang
// modul ini: idempotensi webhook (webhook_events, generik), config gateway terenkripsi
// (payment_gateway_config, termasuk kredensial ElProof), registry app internal (payment_clients),
// dan indeks tipis order_ref→app_id (payment_charge_apps, PLAN.md §9.1.4) — bukan ledger, hanya
// untuk dispatch webhook.
package infrastructure

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/platform/config"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
	uow "github.com/elkasir/api/internal/platform/uow"
)

type apiClient struct {
	mu       sync.RWMutex
	gw       gateway // nil = mode simulasi (tak ada provider aktif)
	provider string  // label provider untuk kolom payments/webhook_events
	// elproofGW adalah dompet TERPISAH, SELALU dicoba dibangun berdampingan dengan gw di atas —
	// bukan dipilih lewat switch Provider yang sama. HANYA dipakai untuk appID AppSubscribe
	// (billing subscription); nil-secara-konten (enabled()==false) berarti mode simulasi untuk
	// jalur ElProof secara independen dari mode simulasi gw Tripay/Midtrans. Lihat PLAN.md §11.
	elproofGW *elproofGateway
	uow       *uow.Manager
	encKey    [32]byte       // AES-256-GCM key, diturunkan dari CONFIG_ENCRYPTION_KEY (§9.1.2)
	baseCfg   config.Payment // config awal dari env, HANYA dipakai utk migrasi satu-kali (§9.1.7)

	consumersMu sync.RWMutex
	consumers   map[string]paymentclient.WebhookConsumer // appID -> consumer, internal saja
}

var (
	_ paymentclient.Client     = (*apiClient)(nil)
	_ paymentclient.Dispatcher = (*apiClient)(nil)
)

// NewClient membuat implementasi paymentclient.Client. encryptionKey adalah nilai mentah
// CONFIG_ENCRYPTION_KEY (panjang bebas — di-hash SHA-256 jadi kunci AES-256 yang valid, §9.1.2).
// Melakukan migrasi env→DB satu-kali (§9.1.7) secara sinkron pada konstruksi: bila
// payment_gateway_config kosong, baris pertama diisi dari cfg (nilai env yang sudah dimuat
// config.Load); setelah itu env TRIPAY_*/MIDTRANS_*/PAYMENT_PROVIDER/PAYMENT_ENV tidak pernah
// dibaca lagi oleh modul ini.
func NewClient(cfg config.Payment, m *uow.Manager, encryptionKey string) paymentclient.Client {
	c := &apiClient{
		uow:       m,
		encKey:    sha256.Sum256([]byte(encryptionKey)),
		baseCfg:   cfg,
		consumers: make(map[string]paymentclient.WebhookConsumer, 2),
	}
	ctx := context.Background()
	if err := c.migrateEnvToDBIfEmpty(ctx); err != nil || c.reload(ctx) != nil {
		// Non-fatal: jatuh ke config env langsung (perilaku pra-Part-2) bila migrasi/DB gagal
		// saat boot — app tetap bisa jalan, superadmin bisa perbaiki lewat Konfigurasi
		// Pembayaran begitu DB tersedia.
		c.setGateway(selectGateway(cfg))
	}
	return c
}

func (c *apiClient) setGateway(gw gateway) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gw = gw
	provider := "midtrans"
	if gw != nil {
		provider = gw.name()
	}
	if provider != "tripay" && provider != "midtrans" {
		provider = "midtrans"
	}
	c.provider = provider
}

func (c *apiClient) activeGateway() (gateway, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.gw, c.provider
}

// setElProof / activeElProof mirror setGateway / activeGateway above for the second, always-
// present ElProof wallet (§11) — guarded by the SAME mutex since both are rebuilt together in
// reload().
func (c *apiClient) setElProof(gw *elproofGateway) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.elproofGW = gw
}

func (c *apiClient) activeElProof() *elproofGateway {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.elproofGW
}

func (c *apiClient) Enabled() bool {
	gw, _ := c.activeGateway()
	return gw != nil
}

// ActiveProviderName implements paymentclient.Client.
func (c *apiClient) ActiveProviderName() string {
	_, provider := c.activeGateway()
	return provider
}

// QuoteFee mengembalikan biaya gateway untuk `amount`. 0 saat mode simulasi (gateway nil).
func (c *apiClient) QuoteFee(ctx context.Context, amount int64) (int64, error) {
	gw, _ := c.activeGateway()
	if gw == nil {
		return 0, nil
	}
	return gw.quoteFee(ctx, amount)
}

// CreateCharge adalah wrapper QRIS-khusus di atas CreateChannelCharge (§9.1.8) — perilaku tak
// berubah dari sebelum Part 2, hanya menambah appID.
func (c *apiClient) CreateCharge(ctx context.Context, appID, storeID, orderID string, amount int64) (paymentclient.Charge, error) {
	return c.CreateChannelCharge(ctx, appID, storeID, orderID, amount, paymentclient.ChannelQRIS, paymentclient.ChannelOptions{})
}

// CreateChannelCharge membuat tagihan lewat gateway yang sesuai UNTUK appID ini (§11): appID
// AppSubscribe SELALU lewat ElProof (dompet TERPISAH khusus subscription billing — lihat
// elproof.go); appID lain (mis. AppSelfOrder) tetap lewat SATU wallet Tripay/Midtrans aktif
// seperti sebelumnya (§9.1.8). storeID tidak dipakai gateway manapun (tak ada baris ledger yang
// ditulis modul ini) — tetap bagian kontrak karena provider lain di masa depan mungkin butuh
// identitas merchant per-tenant. Setiap charge dicatat di indeks order_ref→appID (§9.1.4),
// TERLEPAS dari mode simulasi atau tidak, supaya dispatch webhook tetap konsisten diuji di kedua
// mode.
func (c *apiClient) CreateChannelCharge(ctx context.Context, appID, storeID, orderID string, amount int64, channel paymentclient.Channel, opts paymentclient.ChannelOptions) (paymentclient.Charge, error) {
	var charge paymentclient.Charge
	if appID == paymentclient.AppSubscribe {
		elp := c.activeElProof()
		if elp == nil || !elp.enabled() {
			charge = paymentclient.Charge{Channel: channel, Provider: "elproof", Simulated: true}
		} else {
			res, err := elp.createCharge(ctx, orderID, amount, channel, opts)
			if err != nil {
				return paymentclient.Charge{}, err
			}
			charge = paymentclient.Charge{
				Channel: channel, QRString: res.QRString, QRImageURL: res.QRImageURL,
				VANumber: res.VANumber, VABankCode: res.VABankCode, ProviderRef: res.Ref, Provider: "elproof",
			}
		}
	} else {
		gw, provider := c.activeGateway()
		if gw == nil {
			charge = paymentclient.Charge{Channel: channel, Provider: provider, Simulated: true}
		} else {
			res, err := gw.createCharge(ctx, orderID, amount, channel, opts)
			if err != nil {
				return paymentclient.Charge{}, err
			}
			charge = paymentclient.Charge{
				Channel: channel, QRString: res.QRString, QRImageURL: res.QRImageURL,
				VANumber: res.VANumber, VABankCode: res.VABankCode, ProviderRef: res.Ref, Provider: provider,
			}
		}
	}

	if err := c.uow.Q(ctx).CreateChargeApp(ctx, sqlcgen.CreateChargeAppParams{
		OrderRef: orderID, AppID: appID,
	}); err != nil {
		return paymentclient.Charge{}, fmt.Errorf("payment: gagal mencatat indeks app untuk charge: %w", err)
	}
	return charge, nil
}

// CheckStatus adalah pull-based status check, independen webhook (§9.1.8). appID menentukan
// gateway mana yang diperiksa (sama seperti CreateChannelCharge). PENTING: makna `ref` berbeda
// per gateway — untuk AppSubscribe (ElProof) ini adalah ORDER REF yang sama dikirim ke
// CreateChannelCharge (invoice ID subscription itu sendiri); untuk appID lain (Tripay/Midtrans)
// ini adalah providerRef milik gateway tersebut.
func (c *apiClient) CheckStatus(ctx context.Context, appID, ref string) (paymentclient.ChargeStatus, error) {
	if appID == paymentclient.AppSubscribe {
		elp := c.activeElProof()
		if elp == nil || !elp.enabled() {
			return paymentclient.ChargeStatus{}, errors.New("payment: ElProof nonaktif (mode simulasi)")
		}
		return elp.checkStatus(ctx, ref)
	}
	gw, _ := c.activeGateway()
	if gw == nil {
		return paymentclient.ChargeStatus{}, errors.New("payment: gateway nonaktif (mode simulasi)")
	}
	return gw.checkStatus(ctx, ref)
}

// VerifyWebhook: skema verifikasi spesifik provider hidup di gateway aktif (bukan di selforder).
func (c *apiClient) VerifyWebhook(header http.Header, body []byte) bool {
	gw, _ := c.activeGateway()
	return gw != nil && gw.verifyWebhook(header, body)
}

func (c *apiClient) ParseWebhook(body []byte) (paymentclient.WebhookEvent, error) {
	gw, _ := c.activeGateway()
	if gw == nil {
		return paymentclient.WebhookEvent{}, paymentclient.ErrInvalidPayload
	}
	return gw.parseWebhook(body)
}

// VerifyElProofWebhook/ParseElProofWebhook implement paymentclient.Client — the ElProof-specific
// counterpart to VerifyWebhook/ParseWebhook above (§11), always available regardless of which
// Tripay/Midtrans provider is currently active.
func (c *apiClient) VerifyElProofWebhook(header http.Header, body []byte) bool {
	elp := c.activeElProof()
	return elp != nil && elp.verifyWebhook(header, body)
}

func (c *apiClient) ParseElProofWebhook(body []byte) (paymentclient.WebhookEvent, error) {
	elp := c.activeElProof()
	if elp == nil {
		return paymentclient.WebhookEvent{}, paymentclient.ErrInvalidPayload
	}
	return elp.parseWebhook(body)
}

func (c *apiClient) WebhookSeen(ctx context.Context, provider, eventID string) (bool, error) {
	_, err := c.uow.Q(ctx).GetWebhookEvent(ctx, sqlcgen.GetWebhookEventParams{Provider: provider, EventID: eventID})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (c *apiClient) MarkWebhookSeen(ctx context.Context, provider, eventID string) error {
	return c.uow.Q(ctx).CreateWebhookEvent(ctx, sqlcgen.CreateWebhookEventParams{
		ID: id.New(), Provider: provider, EventID: eventID,
	})
}

// ── Dispatcher (composition root + presentation only, §9.1.5 — not part of Client) ───────────

// RegisterConsumer implements paymentclient.Dispatcher.
func (c *apiClient) RegisterConsumer(appID string, consumer paymentclient.WebhookConsumer) {
	c.consumersMu.Lock()
	defer c.consumersMu.Unlock()
	c.consumers[appID] = consumer
}

// Dispatch implements paymentclient.Dispatcher — resolves which registered in-process consumer
// owns ev.OrderRef (via the order_ref→app_id index written at charge-creation time, §9.1.4) and
// calls ApplyWebhookEvent on it directly, synchronously. Both remaining callers of this
// (/webhooks/payment for Tripay/Midtrans, /webhooks/payment/elproof for ElProof) already
// normalize their provider's callback into the same paymentclient.WebhookEvent shape before
// calling this — Dispatch itself no longer needs to know which gateway an event came from.
// Simplified after Part 3's removal (§11): there is no `kind=external` case anymore, so the
// payment_clients registry lookup + outbound relay branch that used to live here is gone —
// every registered app is `kind=internal` now.
func (c *apiClient) Dispatch(ctx context.Context, ev paymentclient.WebhookEvent) error {
	appID, err := c.uow.Q(ctx).GetChargeApp(ctx, ev.OrderRef)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("payment: order_ref %q tidak dikenali (tidak pernah dibuat lewat CreateCharge)", ev.OrderRef)
	}
	if err != nil {
		return err
	}

	c.consumersMu.RLock()
	consumer, ok := c.consumers[appID]
	c.consumersMu.RUnlock()
	if !ok {
		return fmt.Errorf("payment: tidak ada consumer terdaftar untuk app %q", appID)
	}
	return consumer.ApplyWebhookEvent(ctx, ev)
}

// ── Config gateway (§9.1.2/§9.1.6) ────────────────────────────────────────────────────────────

func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "••••"
	}
	return "••••" + s[len(s)-4:]
}

// GetConfig implements paymentclient.Client.
func (c *apiClient) GetConfig(ctx context.Context) (paymentclient.GatewayConfig, error) {
	row, err := c.uow.Q(ctx).GetPaymentGatewayConfig(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return paymentclient.GatewayConfig{}, nil
	}
	if err != nil {
		return paymentclient.GatewayConfig{}, err
	}
	tripayAPIKey, _ := c.decryptField(row.TripayApiKeyEnc)
	tripayPrivateKey, _ := c.decryptField(row.TripayPrivateKeyEnc)
	tripayMerchantCode, _ := c.decryptField(row.TripayMerchantCodeEnc)
	midtransServerKey, _ := c.decryptField(row.MidtransServerKeyEnc)
	elproofSecret, _ := c.decryptField(row.ElproofSecretEnc)
	return paymentclient.GatewayConfig{
		Provider:                row.Provider,
		Sandbox:                 row.Sandbox,
		TripayAPIKeyMasked:      maskSecret(tripayAPIKey),
		TripayPrivateKeyMasked:  maskSecret(tripayPrivateKey),
		TripayMerchantCode:      tripayMerchantCode,
		TripayMethod:            row.TripayMethod,
		MidtransServerKeyMasked: maskSecret(midtransServerKey),
		ElProofAppID:            row.ElproofAppID.String,
		ElProofSecretMasked:     maskSecret(elproofSecret),
		ElProofBaseURL:          row.ElproofBaseUrl,
	}, nil
}

// UpdateConfig implements paymentclient.Client. Field pointer nil = biarkan nilai terenkripsi
// yang sudah ada (form write-only aman, §9.1.2). Setelah menulis, gateway aktif di-rebuild
// LANGSUNG (§9.1.6) — tidak ada cache, tidak ada poll.
func (c *apiClient) UpdateConfig(ctx context.Context, in paymentclient.UpdateGatewayConfigInput) (paymentclient.GatewayConfig, error) {
	if in.Provider != "" && in.Provider != "tripay" && in.Provider != "midtrans" {
		return paymentclient.GatewayConfig{}, httpx.Validation("Provider harus 'tripay' atau 'midtrans'.")
	}
	q := c.uow.Q(ctx)
	existing, err := q.GetPaymentGatewayConfig(ctx)
	isNew := errors.Is(err, sql.ErrNoRows)
	if err != nil && !isNew {
		return paymentclient.GatewayConfig{}, err
	}

	tripayAPIKeyEnc, err := c.resolveEncField(existing.TripayApiKeyEnc, in.TripayAPIKey)
	if err != nil {
		return paymentclient.GatewayConfig{}, err
	}
	tripayPrivateKeyEnc, err := c.resolveEncField(existing.TripayPrivateKeyEnc, in.TripayPrivateKey)
	if err != nil {
		return paymentclient.GatewayConfig{}, err
	}
	tripayMerchantCodeEnc, err := c.resolveEncField(existing.TripayMerchantCodeEnc, in.TripayMerchantCode)
	if err != nil {
		return paymentclient.GatewayConfig{}, err
	}
	midtransServerKeyEnc, err := c.resolveEncField(existing.MidtransServerKeyEnc, in.MidtransServerKey)
	if err != nil {
		return paymentclient.GatewayConfig{}, err
	}
	elproofSecretEnc, err := c.resolveEncField(existing.ElproofSecretEnc, in.ElProofSecret)
	if err != nil {
		return paymentclient.GatewayConfig{}, err
	}
	method := strings.TrimSpace(in.TripayMethod)
	if method == "" {
		method = firstNonEmpty(existing.TripayMethod, "QRIS")
	}
	elproofAppID := existing.ElproofAppID
	if in.ElProofAppID != nil {
		elproofAppID = sql.NullString{String: strings.TrimSpace(*in.ElProofAppID), Valid: strings.TrimSpace(*in.ElProofAppID) != ""}
	}
	elproofBaseURL := strings.TrimSpace(in.ElProofBaseURL)
	if elproofBaseURL == "" {
		elproofBaseURL = firstNonEmpty(existing.ElproofBaseUrl, elproofDefaultBaseURL)
	}

	if isNew {
		if err := q.InsertPaymentGatewayConfig(ctx, sqlcgen.InsertPaymentGatewayConfigParams{
			ID: id.New(), Provider: in.Provider, Sandbox: in.Sandbox,
			TripayApiKeyEnc: tripayAPIKeyEnc, TripayPrivateKeyEnc: tripayPrivateKeyEnc,
			TripayMerchantCodeEnc: tripayMerchantCodeEnc, TripayMethod: method,
			MidtransServerKeyEnc: midtransServerKeyEnc,
		}); err != nil {
			return paymentclient.GatewayConfig{}, err
		}
		existing, err = q.GetPaymentGatewayConfig(ctx)
		if err != nil {
			return paymentclient.GatewayConfig{}, err
		}
	}
	if err := q.UpdatePaymentGatewayConfig(ctx, sqlcgen.UpdatePaymentGatewayConfigParams{
		Provider: in.Provider, Sandbox: in.Sandbox,
		TripayApiKeyEnc: tripayAPIKeyEnc, TripayPrivateKeyEnc: tripayPrivateKeyEnc,
		TripayMerchantCodeEnc: tripayMerchantCodeEnc, TripayMethod: method,
		MidtransServerKeyEnc: midtransServerKeyEnc, ID: existing.ID,
		ElproofAppID: elproofAppID, ElproofSecretEnc: elproofSecretEnc, ElproofBaseUrl: elproofBaseURL,
	}); err != nil {
		return paymentclient.GatewayConfig{}, err
	}

	if err := c.reload(ctx); err != nil {
		return paymentclient.GatewayConfig{}, fmt.Errorf("payment: config tersimpan tapi gagal memuat ulang gateway: %w", err)
	}
	return c.GetConfig(ctx)
}

// resolveEncField: pointer nil → pertahankan nilai lama; non-nil → enkripsi nilai baru
// (termasuk string kosong eksplisit, utk mengosongkan secara sengaja).
func (c *apiClient) resolveEncField(existing sql.NullString, next *string) (sql.NullString, error) {
	if next == nil {
		return existing, nil
	}
	return c.encryptField(*next)
}

func (c *apiClient) encryptField(plain string) (sql.NullString, error) {
	if plain == "" {
		return sql.NullString{}, nil
	}
	enc, err := encryptAESGCM(c.encKey, plain)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: enc, Valid: true}, nil
}

func (c *apiClient) decryptField(enc sql.NullString) (string, error) {
	if !enc.Valid || enc.String == "" {
		return "", nil
	}
	return decryptAESGCM(c.encKey, enc.String)
}

// reload rebuilds the active gateway from the DB-stored config (§9.1.6) — no cache, called
// explicitly right after a successful UpdateConfig write, and once at construction.
func (c *apiClient) reload(ctx context.Context) error {
	row, err := c.uow.Q(ctx).GetPaymentGatewayConfig(ctx)
	if err != nil {
		return err
	}
	tripayAPIKey, err := c.decryptField(row.TripayApiKeyEnc)
	if err != nil {
		return err
	}
	tripayPrivateKey, err := c.decryptField(row.TripayPrivateKeyEnc)
	if err != nil {
		return err
	}
	tripayMerchantCode, err := c.decryptField(row.TripayMerchantCodeEnc)
	if err != nil {
		return err
	}
	midtransServerKey, err := c.decryptField(row.MidtransServerKeyEnc)
	if err != nil {
		return err
	}
	elproofSecret, err := c.decryptField(row.ElproofSecretEnc)
	if err != nil {
		return err
	}
	cfg := c.baseCfg // CallbackURL & BaseURL derivation (PublicBaseURL/env) stay as boot-time infra config
	cfg.Provider = row.Provider
	cfg.Sandbox = row.Sandbox
	cfg.Tripay.APIKey = tripayAPIKey
	cfg.Tripay.PrivateKey = tripayPrivateKey
	cfg.Tripay.MerchantCode = tripayMerchantCode
	cfg.Tripay.Method = row.TripayMethod
	cfg.Midtrans.ServerKey = midtransServerKey
	c.setGateway(selectGateway(cfg))
	// ElProof (§11): dompet TERPISAH, dibangun berdampingan — TIDAK lewat selectGateway/Provider
	// switch di atas. enabled()==false (appID/secret kosong) berarti mode simulasi untuk jalur
	// AppSubscribe, independen dari status Provider Tripay/Midtrans.
	c.setElProof(newElproof(row.ElproofAppID.String, elproofSecret, row.ElproofBaseUrl))
	return nil
}

// migrateEnvToDBIfEmpty implements §9.1.7: one-time env→DB migration. If the config table
// already has a row, this is a no-op — env vars are never consulted again after the first boot
// that finds the table empty.
func (c *apiClient) migrateEnvToDBIfEmpty(ctx context.Context) error {
	q := c.uow.Q(ctx)
	n, err := q.CountPaymentGatewayConfig(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	tripayAPIKeyEnc, err := c.encryptField(c.baseCfg.Tripay.APIKey)
	if err != nil {
		return err
	}
	tripayPrivateKeyEnc, err := c.encryptField(c.baseCfg.Tripay.PrivateKey)
	if err != nil {
		return err
	}
	tripayMerchantCodeEnc, err := c.encryptField(c.baseCfg.Tripay.MerchantCode)
	if err != nil {
		return err
	}
	midtransServerKeyEnc, err := c.encryptField(c.baseCfg.Midtrans.ServerKey)
	if err != nil {
		return err
	}
	method := c.baseCfg.Tripay.Method
	if method == "" {
		method = "QRIS"
	}
	return q.InsertPaymentGatewayConfig(ctx, sqlcgen.InsertPaymentGatewayConfigParams{
		ID: id.New(), Provider: c.baseCfg.Provider, Sandbox: c.baseCfg.Sandbox,
		TripayApiKeyEnc: tripayAPIKeyEnc, TripayPrivateKeyEnc: tripayPrivateKeyEnc,
		TripayMerchantCodeEnc: tripayMerchantCodeEnc, TripayMethod: method,
		MidtransServerKeyEnc: midtransServerKeyEnc,
	})
}

// ── AES-256-GCM helpers (stdlib only, no third-party crypto dep — §9.1.2) ────────────────────

func encryptAESGCM(key [32]byte, plain string) (string, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptAESGCM(key [32]byte, encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("payment: ciphertext config terlalu pendek")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
