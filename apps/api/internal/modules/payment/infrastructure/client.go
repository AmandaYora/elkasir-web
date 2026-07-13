// apiClient mengimplementasikan paymentclient.Client (dan paymentclient.Dispatcher, hanya
// dipakai composition root + presentation) di atas SATU gateway aktif (lihat gateway.go) +
// konfigurasi gateway yang di-DB-kan (§9.1.2) + registry app (§9.1.3). Modul ini SENGAJA tidak
// mencatat baris ledger bisnis apa pun (mis. tabel self-order/subscription) — itu tanggung
// jawab masing-masing pemanggil (lihat paymentclient.Charge.Provider). State yang dipegang
// modul ini: idempotensi webhook (webhook_events, generik), config gateway terenkripsi
// (payment_gateway_config), registry app (payment_clients), dan indeks tipis order_ref→app_id
// (payment_charge_apps, PLAN.md §9.1.4) — bukan ledger, hanya untuk dispatch webhook.
package infrastructure

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/platform/config"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
	"github.com/elkasir/api/internal/platform/security"
	uow "github.com/elkasir/api/internal/platform/uow"
)

type apiClient struct {
	mu       sync.RWMutex
	gw       gateway // nil = mode simulasi (tak ada provider aktif)
	provider string  // label provider untuk kolom payments/webhook_events
	uow      *uow.Manager
	encKey   [32]byte       // AES-256-GCM key, diturunkan dari CONFIG_ENCRYPTION_KEY (§9.1.2)
	baseCfg  config.Payment // config awal dari env, HANYA dipakai utk migrasi satu-kali (§9.1.7)

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

func (c *apiClient) Enabled() bool {
	gw, _ := c.activeGateway()
	return gw != nil
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

// CreateChannelCharge membuat tagihan lewat gateway aktif pada kanal manapun yang didukung
// (§9.1.8). storeID tidak dipakai gateway (tak ada baris ledger yang ditulis modul ini) — tetap
// bagian kontrak karena provider lain di masa depan mungkin butuh identitas merchant per-tenant.
// Setiap charge dicatat di indeks order_ref→appID (§9.1.4), TERLEPAS dari mode simulasi atau
// tidak, supaya dispatch webhook tetap konsisten diuji di kedua mode.
func (c *apiClient) CreateChannelCharge(ctx context.Context, appID, storeID, orderID string, amount int64, channel paymentclient.Channel, opts paymentclient.ChannelOptions) (paymentclient.Charge, error) {
	gw, provider := c.activeGateway()

	var charge paymentclient.Charge
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

	providerRef := sql.NullString{}
	if charge.ProviderRef != "" {
		providerRef = sql.NullString{String: charge.ProviderRef, Valid: true}
	}
	if err := c.uow.Q(ctx).CreateChargeApp(ctx, sqlcgen.CreateChargeAppParams{
		OrderRef: orderID, AppID: appID, ProviderRef: providerRef,
	}); err != nil {
		return paymentclient.Charge{}, fmt.Errorf("payment: gagal mencatat indeks app untuk charge: %w", err)
	}
	return charge, nil
}

// ListChannels melaporkan kanal yang aktif di akun gateway saat ini (§9.1.8). Kosong (bukan
// error) dalam mode simulasi.
func (c *apiClient) ListChannels(ctx context.Context) ([]paymentclient.ChannelInfo, error) {
	gw, _ := c.activeGateway()
	if gw == nil {
		return nil, nil
	}
	return gw.listChannels(ctx)
}

// CheckStatus adalah pull-based status check, independen webhook (§9.1.8).
func (c *apiClient) CheckStatus(ctx context.Context, providerRef string) (paymentclient.ChargeStatus, error) {
	gw, _ := c.activeGateway()
	if gw == nil {
		return paymentclient.ChargeStatus{}, errors.New("payment: gateway nonaktif (mode simulasi)")
	}
	return gw.checkStatus(ctx, providerRef)
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

func (c *apiClient) WebhookSeen(ctx context.Context, eventID string) (bool, error) {
	_, provider := c.activeGateway()
	_, err := c.uow.Q(ctx).GetWebhookEvent(ctx, sqlcgen.GetWebhookEventParams{Provider: provider, EventID: eventID})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (c *apiClient) MarkWebhookSeen(ctx context.Context, eventID string) error {
	_, provider := c.activeGateway()
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

// Dispatch implements paymentclient.Dispatcher — resolves which registered app owns ev.OrderRef
// (via the order_ref→app_id index written at charge-creation time, §9.1.4) and routes the event
// to it. Replaces the old "sub_"-prefix sniffing that used to live in internal/app/webhook.go.
//
// Two branches (§10.1.10, Part 3): a `kind=internal` app (self-order, subscription) is an
// in-process Go consumer — call ApplyWebhookEvent directly, synchronously, exactly as before
// Part 3. A `kind=external` app has no in-process consumer at all — instead, spawn a
// fire-and-forget goroutine that relays a signed payload to its callback_url and return
// immediately; the relay's own outcome never blocks or affects Elkasir's ack to the gateway.
func (c *apiClient) Dispatch(ctx context.Context, ev paymentclient.WebhookEvent) error {
	appID, err := c.uow.Q(ctx).GetChargeApp(ctx, ev.OrderRef)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("payment: order_ref %q tidak dikenali (tidak pernah dibuat lewat CreateCharge)", ev.OrderRef)
	}
	if err != nil {
		return err
	}

	row, err := c.uow.Q(ctx).GetPaymentClientByAppID(ctx, appID)
	if err != nil {
		return fmt.Errorf("payment: app %q tidak ditemukan di registry: %w", appID, err)
	}

	if row.Kind == sqlcgen.PaymentClientsKindExternal {
		go c.relayWebhook(context.WithoutCancel(ctx), row, ev)
		return nil
	}

	c.consumersMu.RLock()
	consumer, ok := c.consumers[appID]
	c.consumersMu.RUnlock()
	if !ok {
		return fmt.Errorf("payment: tidak ada consumer terdaftar untuk app %q", appID)
	}
	return consumer.ApplyWebhookEvent(ctx, ev)
}

// relayWebhookPayload is the JSON body relayed to an external app's callback_url — documented in
// docs/EXTERNAL_PAYMENT_API.md (§10.2 EB5).
type relayWebhookPayload struct {
	EventID   string `json:"eventId"`
	OrderRef  string `json:"orderRef"`
	Paid      bool   `json:"paid"`
	Timestamp int64  `json:"timestamp"`
}

// relayWebhook signs and POSTs a webhook event to an external app's callback_url — exactly ONE
// attempt (§10.1.10), never retried; the app is expected to fall back to
// GET /external/payments/charges/{orderRef}/status if a relay is ever lost. Runs in its own
// goroutine (see Dispatch) with a context already detached from the original request
// (context.WithoutCancel) — same fire-and-forget shape as withdrawal's email notification.
func (c *apiClient) relayWebhook(ctx context.Context, row sqlcgen.PaymentClient, ev paymentclient.WebhookEvent) {
	if !row.CallbackUrl.Valid || strings.TrimSpace(row.CallbackUrl.String) == "" {
		slog.Warn("payment: app eksternal tanpa callback_url, relay dilewati", "appId", row.AppID)
		return
	}
	if !row.SecretEnc.Valid {
		slog.Warn("payment: app eksternal tanpa secret_enc, relay dibatalkan", "appId", row.AppID)
		return
	}
	secret, err := decryptAESGCM(c.encKey, row.SecretEnc.String)
	if err != nil {
		slog.Warn("payment: gagal dekripsi secret utk relay webhook", "appId", row.AppID, "err", err)
		return
	}

	body, err := json.Marshal(relayWebhookPayload{
		EventID: ev.EventID, OrderRef: ev.OrderRef, Paid: ev.Paid, Timestamp: time.Now().Unix(),
	})
	if err != nil {
		slog.Warn("payment: gagal marshal payload relay webhook", "appId", row.AppID, "err", err)
		return
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, row.CallbackUrl.String, bytes.NewReader(body))
	if err != nil {
		slog.Warn("payment: gagal membuat request relay webhook", "appId", row.AppID, "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Elkasir-Signature", signature)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		slog.Warn("payment: relay webhook gagal terkirim", "appId", row.AppID, "callbackUrl", row.CallbackUrl.String, "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Warn("payment: relay webhook ditolak penerima", "appId", row.AppID, "status", resp.StatusCode)
	}
}

// ResolveApp implements paymentclient.Dispatcher (§10.2 EB2).
func (c *apiClient) ResolveApp(ctx context.Context, rowID string) (paymentclient.AppInfo, error) {
	row, err := c.uow.Q(ctx).GetPaymentClientByID(ctx, rowID)
	if err != nil {
		return paymentclient.AppInfo{}, err
	}
	return toAppInfo(row), nil
}

// ResolveCharge implements paymentclient.Dispatcher (§10.2 EB2).
func (c *apiClient) ResolveCharge(ctx context.Context, orderRef string) (string, string, error) {
	row, err := c.uow.Q(ctx).GetChargeAppByOrderRef(ctx, orderRef)
	if err != nil {
		return "", "", err
	}
	return row.AppID, row.ProviderRef.String, nil
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
	return paymentclient.GatewayConfig{
		Provider:                row.Provider,
		Sandbox:                 row.Sandbox,
		TripayAPIKeyMasked:      maskSecret(tripayAPIKey),
		TripayPrivateKeyMasked:  maskSecret(tripayPrivateKey),
		TripayMerchantCode:      tripayMerchantCode,
		TripayMethod:            row.TripayMethod,
		MidtransServerKeyMasked: maskSecret(midtransServerKey),
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
	method := strings.TrimSpace(in.TripayMethod)
	if method == "" {
		method = firstNonEmpty(existing.TripayMethod, "QRIS")
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
	} else {
		if err := q.UpdatePaymentGatewayConfig(ctx, sqlcgen.UpdatePaymentGatewayConfigParams{
			Provider: in.Provider, Sandbox: in.Sandbox,
			TripayApiKeyEnc: tripayAPIKeyEnc, TripayPrivateKeyEnc: tripayPrivateKeyEnc,
			TripayMerchantCodeEnc: tripayMerchantCodeEnc, TripayMethod: method,
			MidtransServerKeyEnc: midtransServerKeyEnc, ID: existing.ID,
		}); err != nil {
			return paymentclient.GatewayConfig{}, err
		}
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
	cfg := c.baseCfg // CallbackURL & BaseURL derivation (PublicBaseURL/env) stay as boot-time infra config
	cfg.Provider = row.Provider
	cfg.Sandbox = row.Sandbox
	cfg.Tripay.APIKey = tripayAPIKey
	cfg.Tripay.PrivateKey = tripayPrivateKey
	cfg.Tripay.MerchantCode = tripayMerchantCode
	cfg.Tripay.Method = row.TripayMethod
	cfg.Midtrans.ServerKey = midtransServerKey
	c.setGateway(selectGateway(cfg))
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

// ── Registry app (§9.1.3) ──────────────────────────────────────────────────────────────────

func toAppInfo(row sqlcgen.PaymentClient) paymentclient.AppInfo {
	return paymentclient.AppInfo{
		ID: row.ID, AppID: row.AppID, Name: row.Name, Kind: string(row.Kind),
		CallbackURL: row.CallbackUrl.String, Status: string(row.Status), CreatedAt: row.CreatedAt,
	}
}

// ListApps implements paymentclient.Client.
func (c *apiClient) ListApps(ctx context.Context) ([]paymentclient.AppInfo, error) {
	rows, err := c.uow.Q(ctx).ListPaymentClients(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]paymentclient.AppInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAppInfo(r))
	}
	return out, nil
}

// CreateApp implements paymentclient.Client — always creates a kind='external' row (the two
// kind='internal' rows are seeded once by migration 000019, never created through this path,
// §9.1.3). Secret is returned ONLY here, in plaintext, once — never again afterward.
func (c *apiClient) CreateApp(ctx context.Context, in paymentclient.CreateAppInput) (paymentclient.CreateAppResult, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return paymentclient.CreateAppResult{}, httpx.Validation("Nama aplikasi wajib diisi.")
	}
	secret, err := generateSecret()
	if err != nil {
		return paymentclient.CreateAppResult{}, err
	}
	hash, err := security.HashPassword(secret)
	if err != nil {
		return paymentclient.CreateAppResult{}, err
	}
	// secret_enc: SAMA plaintext, disimpan reversibel (bukan secret kedua) — hanya dipakai saat
	// menandatangani relay webhook keluar, karena bcrypt (secret_hash) tak bisa dibalik (§10.1.6).
	enc, err := c.encryptField(secret)
	if err != nil {
		return paymentclient.CreateAppResult{}, err
	}
	uid := id.New()
	appID := generateAppID(name)
	callback := sql.NullString{}
	if cb := strings.TrimSpace(in.CallbackURL); cb != "" {
		callback = sql.NullString{String: cb, Valid: true}
	}
	if err := c.uow.Q(ctx).CreatePaymentClient(ctx, sqlcgen.CreatePaymentClientParams{
		ID: uid, AppID: appID, Name: name, SecretHash: sql.NullString{String: hash, Valid: true},
		SecretEnc: enc, Kind: sqlcgen.PaymentClientsKindExternal, CallbackUrl: callback,
	}); err != nil {
		return paymentclient.CreateAppResult{}, err
	}
	row, err := c.uow.Q(ctx).GetPaymentClientByID(ctx, uid)
	if err != nil {
		return paymentclient.CreateAppResult{}, err
	}
	return paymentclient.CreateAppResult{AppInfo: toAppInfo(row), Secret: secret}, nil
}

// ResetAppSecret implements paymentclient.Client — kind='internal' rows are rejected by the
// underlying query's own `AND kind='external'` guard (0 rows affected → NotFound here).
func (c *apiClient) ResetAppSecret(ctx context.Context, appRowID string) (string, error) {
	secret, err := generateSecret()
	if err != nil {
		return "", err
	}
	hash, err := security.HashPassword(secret)
	if err != nil {
		return "", err
	}
	enc, err := c.encryptField(secret) // same plaintext, reversible copy — §10.1.6
	if err != nil {
		return "", err
	}
	n, err := c.uow.Q(ctx).SetPaymentClientSecret(ctx, sqlcgen.SetPaymentClientSecretParams{
		SecretHash: sql.NullString{String: hash, Valid: true}, SecretEnc: enc, ID: appRowID,
	})
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", httpx.NotFound("Aplikasi tidak ditemukan, atau merupakan aplikasi internal bawaan sistem.")
	}
	return secret, nil
}

// SetAppStatus implements paymentclient.Client — same internal-row guard as ResetAppSecret
// (§9.1.3: ELKASIR-SELFORDER/ELKASIR-SUBSCRIBE can never be deactivated through this path).
func (c *apiClient) SetAppStatus(ctx context.Context, appRowID, status string) error {
	if status != "active" && status != "inactive" {
		return httpx.Validation("Status harus 'active' atau 'inactive'.")
	}
	n, err := c.uow.Q(ctx).SetPaymentClientStatus(ctx, sqlcgen.SetPaymentClientStatusParams{
		Status: sqlcgen.PaymentClientsStatus(status), ID: appRowID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return httpx.NotFound("Aplikasi tidak ditemukan, atau merupakan aplikasi internal bawaan sistem.")
	}
	return nil
}

func generateSecret() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// generateAppID derives a readable, unique app_id from the given name (slug + random suffix) —
// collisions are astronomically unlikely (4 random bytes) and the unique index on app_id is the
// actual safety net regardless.
func generateAppID(name string) string {
	slug := strings.ToUpper(strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r - 32
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		default:
			return '-'
		}
	}, name))
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "APP"
	}
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	return slug + "-" + strings.ToUpper(hex.EncodeToString(suffix))
}
