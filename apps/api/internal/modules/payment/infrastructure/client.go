// Package infrastructure implements paymentclient.Client: membuat charge QRIS
// (Xendit/simulasi), mencatat baris payments, serta verifikasi/parse/idempotensi webhook
// (tabel webhook_events). Semua akses DB lewat uow.Q(ctx) sehingga konsisten dengan
// transaksi aktif (bila ada).
package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/platform/config"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/id"
	uow "github.com/elkasir/api/internal/platform/uow"
)

const providerXendit = "xendit"

type apiClient struct {
	xen *xenditClient
	uow *uow.Manager
}

// NewClient membuat implementasi paymentclient.Client.
func NewClient(cfg config.Xendit, m *uow.Manager) paymentclient.Client {
	return &apiClient{xen: newXendit(cfg), uow: m}
}

var _ paymentclient.Client = (*apiClient)(nil)

func (c *apiClient) Enabled() bool { return c.xen.enabled() }

// VerifyWebhook: skema verifikasi spesifik provider hidup DI SINI (bukan di selforder).
// Xendit memakai header statis x-callback-token; provider lain bisa memakai signature body.
func (c *apiClient) VerifyWebhook(header http.Header, _ []byte) bool {
	return c.xen.verifyWebhook(header.Get("x-callback-token"))
}

// CreateCharge membuat tagihan QRIS untuk self-order dan mencatat baris payments.
// Pencatatan payments bersifat best-effort (selaras perilaku lama) — webhook adalah
// sumber kebenaran status bayar.
func (c *apiClient) CreateCharge(ctx context.Context, storeID, orderID string, amount int64) (paymentclient.Charge, error) {
	if !c.xen.enabled() {
		c.recordPayment(ctx, storeID, orderID, amount, sql.NullString{})
		return paymentclient.Charge{Simulated: true}, nil
	}
	qr, err := c.xen.createDynamicQR(ctx, orderID, amount)
	if err != nil {
		return paymentclient.Charge{}, err
	}
	c.recordPayment(ctx, storeID, orderID, amount, sql.NullString{String: qr.ID, Valid: true})
	return paymentclient.Charge{QRString: qr.QRString, ProviderRef: qr.ID}, nil
}

func (c *apiClient) recordPayment(ctx context.Context, storeID, orderID string, amount int64, providerRef sql.NullString) {
	_ = c.uow.Q(ctx).CreatePayment(ctx, sqlcgen.CreatePaymentParams{
		ID: id.New(), StoreID: storeID, SelfOrderID: orderID,
		ProviderRef: providerRef, Amount: amount,
		Status: sqlcgen.PaymentsStatusPending, RawPayload: sql.NullString{},
	})
}

func (c *apiClient) WebhookSeen(ctx context.Context, eventID string) (bool, error) {
	_, err := c.uow.Q(ctx).GetWebhookEvent(ctx, sqlcgen.GetWebhookEventParams{Provider: providerXendit, EventID: eventID})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (c *apiClient) MarkWebhookSeen(ctx context.Context, eventID string) error {
	return c.uow.Q(ctx).CreateWebhookEvent(ctx, sqlcgen.CreateWebhookEventParams{
		ID: id.New(), Provider: providerXendit, EventID: eventID,
	})
}

// ── Parsing payload Xendit ───────────────────────────────────
type webhookPayload struct {
	Event string `json:"event"`
	ID    string `json:"id"`
	Data  struct {
		ID          string `json:"id"`
		QRID        string `json:"qr_id"`
		ReferenceID string `json:"reference_id"`
		Status      string `json:"status"`
		Amount      int64  `json:"amount"`
	} `json:"data"`
}

func (c *apiClient) ParseWebhook(body []byte) (paymentclient.WebhookEvent, error) {
	var p webhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return paymentclient.WebhookEvent{}, paymentclient.ErrInvalidPayload
	}
	return paymentclient.WebhookEvent{
		EventID:  firstNonEmpty(p.ID, p.Data.ID, p.Data.QRID+":"+p.Data.ReferenceID),
		OrderRef: p.Data.ReferenceID,
		Paid:     isPaidStatus(p.Data.Status),
	}, nil
}

func isPaidStatus(s string) bool {
	switch strings.ToUpper(s) {
	case "SUCCEEDED", "COMPLETED", "PAID", "SUCCESS":
		return true
	default:
		return false
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" && v != ":" {
			return v
		}
	}
	return ""
}
