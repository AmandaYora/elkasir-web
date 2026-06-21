// Package infrastructure holds the selforder module's persistence — the Repo, which
// touches ONLY selforder-owned tables (self_orders, self_order_items). Cross-module data
// (produk/shift/tabel/transaksi/pembayaran) is reached by the service via module clients.
//
// All queries go through uow.Q(ctx) so they stay consistent with the active transaction
// (atomic fulfilment). The repo is built from the unit-of-work manager.
package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"github.com/elkasir/api/internal/modules/selforder/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/id"
	"github.com/elkasir/api/internal/platform/uow"
)

type Repo struct{ uow *uow.Manager }

// NewRepo membangun Repo dari unit-of-work manager (tx-aware via uow.Q(ctx)).
func NewRepo(uowMgr *uow.Manager) *Repo { return &Repo{uow: uowMgr} }

// ── Self-order (tabel milik modul) ───────────────────────────
type CreateOrderData struct {
	Order sqlcgen.CreateSelfOrderParams
	Items []domain.OrderItem
}

// CreateOrder menyimpan self-order + item (snapshot) dalam satu transaksi DB (via uow).
func (r *Repo) CreateOrder(ctx context.Context, d CreateOrderData) error {
	return r.uow.Run(ctx, func(ctx context.Context) error {
		q := r.uow.Q(ctx)
		if err := q.CreateSelfOrder(ctx, d.Order); err != nil {
			return err
		}
		for _, it := range d.Items {
			if err := q.CreateSelfOrderItem(ctx, sqlcgen.CreateSelfOrderItemParams{
				ID: id.New(), SelfOrderID: d.Order.ID,
				ProductID:   sql.NullString{String: it.ProductID, Valid: it.ProductID != ""},
				ProductName: it.ProductName, Category: it.Category, Price: it.Price,
				Quantity: it.Quantity, LineTotal: it.LineTotal,
				Note: sql.NullString{String: it.Note, Valid: it.Note != ""},
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repo) Get(ctx context.Context, storeID, soID string) (sqlcgen.SelfOrder, error) {
	return r.uow.Q(ctx).GetSelfOrder(ctx, sqlcgen.GetSelfOrderParams{ID: soID, StoreID: storeID})
}

func (r *Repo) GetByID(ctx context.Context, soID string) (sqlcgen.SelfOrder, error) {
	return r.uow.Q(ctx).GetSelfOrderByID(ctx, soID)
}

func (r *Repo) GetByClaimCode(ctx context.Context, storeID, claim string) (sqlcgen.SelfOrder, error) {
	return r.uow.Q(ctx).GetSelfOrderByClaimCode(ctx, sqlcgen.GetSelfOrderByClaimCodeParams{
		StoreID: storeID, ClaimCode: sql.NullString{String: claim, Valid: true},
	})
}

func (r *Repo) Items(ctx context.Context, soID string) ([]sqlcgen.SelfOrderItem, error) {
	return r.uow.Q(ctx).ListSelfOrderItems(ctx, soID)
}

func (r *Repo) UpdateStatus(ctx context.Context, storeID, soID string, status sqlcgen.SelfOrdersStatus) (int64, error) {
	return r.uow.Q(ctx).UpdateSelfOrderStatus(ctx, sqlcgen.UpdateSelfOrderStatusParams{Status: status, ID: soID, StoreID: storeID})
}

// MarkPaid menautkan self-order ke transaksi + set status. Dipanggil DI DALAM uow.Run
// (fulfilment) sehingga ikut transaksi yang sama dengan kurangi-stok & RecordSale.
func (r *Repo) MarkPaid(ctx context.Context, soID, txID string, status sqlcgen.SelfOrdersStatus) error {
	return r.uow.Q(ctx).MarkSelfOrderPaid(ctx, sqlcgen.MarkSelfOrderPaidParams{
		TransactionID: sql.NullString{String: txID, Valid: true}, Status: status, ID: soID,
	})
}

// ExpireOverdue menandai self-order QRIS yang melewati expires_at sebagai kedaluwarsa.
func (r *Repo) ExpireOverdue(ctx context.Context, storeID string, now time.Time) (int64, error) {
	return r.uow.Q(ctx).ExpireOverdueSelfOrders(ctx, sqlcgen.ExpireOverdueSelfOrdersParams{
		StoreID: storeID, ExpiresAt: sql.NullTime{Time: now, Valid: true},
	})
}

// ListIncoming mengembalikan self-order (opsional difilter status) + total.
func (r *Repo) ListIncoming(ctx context.Context, storeID, status string, limit, offset int) ([]sqlcgen.SelfOrder, int64, error) {
	where := "WHERE store_id = ?"
	args := []any{storeID}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	db := r.uow.DB()
	var total int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM self_orders "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT id, store_id, table_id, status, payment_method, payment_status, claim_code,
		 subtotal, service_charge, gateway_fee, tax, total, customer_note, transaction_id, expires_at, created_at, updated_at
		 FROM self_orders `+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]sqlcgen.SelfOrder, 0, limit)
	for rows.Next() {
		var s sqlcgen.SelfOrder
		if err := rows.Scan(&s.ID, &s.StoreID, &s.TableID, &s.Status, &s.PaymentMethod, &s.PaymentStatus,
			&s.ClaimCode, &s.Subtotal, &s.ServiceCharge, &s.GatewayFee, &s.Tax, &s.Total, &s.CustomerNote, &s.TransactionID, &s.ExpiresAt,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}
