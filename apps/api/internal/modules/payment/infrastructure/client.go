// apiClient mengimplementasikan paymentclient.Client di atas SATU gateway aktif (lihat
// gateway.go). Bagian provider-independent ada di sini: pencatatan tabel payments dan
// idempotensi webhook (tabel webhook_events). Semua akses DB lewat uow.Q(ctx) agar konsisten
// dengan transaksi aktif (bila ada).
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/platform/config"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/id"
	uow "github.com/elkasir/api/internal/platform/uow"
)

type apiClient struct {
	gw       gateway // nil = mode simulasi (tak ada provider aktif)
	provider string  // label provider untuk kolom payments/webhook_events
	uow      *uow.Manager
}

// NewClient membuat implementasi paymentclient.Client dengan gateway aktif terpilih.
func NewClient(cfg config.Payment, m *uow.Manager) paymentclient.Client {
	gw := selectGateway(cfg)
	provider := cfg.ActiveProvider()
	if gw != nil {
		provider = gw.name()
	}
	// payments.provider adalah ENUM — pastikan nilai valid (mode simulasi tanpa provider).
	if provider != "tripay" && provider != "midtrans" {
		provider = "midtrans"
	}
	return &apiClient{gw: gw, provider: provider, uow: m}
}

var _ paymentclient.Client = (*apiClient)(nil)

func (c *apiClient) Enabled() bool { return c.gw != nil }

// QuoteFee mengembalikan biaya gateway untuk `amount`. 0 saat mode simulasi (gateway nil).
func (c *apiClient) QuoteFee(ctx context.Context, amount int64) (int64, error) {
	if c.gw == nil {
		return 0, nil
	}
	return c.gw.quoteFee(ctx, amount)
}

// VerifyWebhook: skema verifikasi spesifik provider hidup di gateway aktif (bukan di selforder).
func (c *apiClient) VerifyWebhook(header http.Header, body []byte) bool {
	return c.gw != nil && c.gw.verifyWebhook(header, body)
}

func (c *apiClient) ParseWebhook(body []byte) (paymentclient.WebhookEvent, error) {
	if c.gw == nil {
		return paymentclient.WebhookEvent{}, paymentclient.ErrInvalidPayload
	}
	return c.gw.parseWebhook(body)
}

// CreateCharge membuat tagihan QRIS untuk self-order dan mencatat baris payments.
// Pencatatan payments bersifat best-effort — webhook adalah sumber kebenaran status bayar.
func (c *apiClient) CreateCharge(ctx context.Context, storeID, orderID string, amount int64) (paymentclient.Charge, error) {
	if c.gw == nil {
		c.recordPayment(ctx, storeID, orderID, amount, sql.NullString{})
		return paymentclient.Charge{Simulated: true}, nil
	}
	qr, err := c.gw.createCharge(ctx, orderID, amount)
	if err != nil {
		return paymentclient.Charge{}, err
	}
	c.recordPayment(ctx, storeID, orderID, amount, sql.NullString{String: qr.Ref, Valid: qr.Ref != ""})
	return paymentclient.Charge{QRString: qr.QRString, QRImageURL: qr.QRImageURL, ProviderRef: qr.Ref}, nil
}

func (c *apiClient) recordPayment(ctx context.Context, storeID, orderID string, amount int64, providerRef sql.NullString) {
	_ = c.uow.Q(ctx).CreatePayment(ctx, sqlcgen.CreatePaymentParams{
		ID: id.New(), StoreID: storeID, SelfOrderID: orderID,
		Provider: sqlcgen.PaymentsProvider(c.provider), ProviderRef: providerRef, Amount: amount,
		Status: sqlcgen.PaymentsStatusPending, RawPayload: sql.NullString{},
	})
}

func (c *apiClient) WebhookSeen(ctx context.Context, eventID string) (bool, error) {
	_, err := c.uow.Q(ctx).GetWebhookEvent(ctx, sqlcgen.GetWebhookEventParams{Provider: c.provider, EventID: eventID})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (c *apiClient) MarkWebhookSeen(ctx context.Context, eventID string) error {
	return c.uow.Q(ctx).CreateWebhookEvent(ctx, sqlcgen.CreateWebhookEventParams{
		ID: id.New(), Provider: c.provider, EventID: eventID,
	})
}
