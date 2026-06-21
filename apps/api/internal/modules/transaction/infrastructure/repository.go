// Package infrastructure holds the transaction module's persistence (sqlc + database/sql)
// and its contract implementation.
//
// Ledger penjualan: pembuatan transaksi POS (Kondisi 1) + listing/detail. Orkestrasi
// atomik (kurangi stok via productclient + RecordSale) ada di service memakai uow;
// pencatatan transaksi sendiri ada di salesclient impl (client.go). Repo ini hanya untuk
// pembacaan & idempotency milik modul transaction.
package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"github.com/elkasir/api/internal/modules/transaction/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

func (r *Repo) Get(ctx context.Context, storeID, txID string) (sqlcgen.Transaction, error) {
	return r.q.GetTransaction(ctx, sqlcgen.GetTransactionParams{ID: txID, StoreID: storeID})
}

func (r *Repo) Items(ctx context.Context, txID string) ([]sqlcgen.TransactionItem, error) {
	return r.q.ListTransactionItems(ctx, txID)
}

func (r *Repo) Idempotency(ctx context.Context, storeID, key string) (sqlcgen.IdempotencyKey, error) {
	return r.q.GetIdempotencyKey(ctx, sqlcgen.GetIdempotencyKeyParams{StoreID: storeID, IdempotencyKey: key})
}

// List mengembalikan transaksi terfilter + total (query dinamis ditulis tangan).
func (r *Repo) List(ctx context.Context, f domain.ListFilter) ([]sqlcgen.Transaction, int64, error) {
	var where strings.Builder
	where.WriteString("WHERE store_id = ?")
	args := []any{f.StoreID}
	if f.Status != "" {
		where.WriteString(" AND status = ?")
		args = append(args, f.Status)
	}
	if f.Source != "" {
		where.WriteString(" AND source = ?")
		args = append(args, f.Source)
	}
	if f.PaymentMethod != "" {
		where.WriteString(" AND payment_method = ?")
		args = append(args, f.PaymentMethod)
	}
	if f.Search != "" {
		where.WriteString(" AND code LIKE ?")
		args = append(args, "%"+f.Search+"%")
	}
	if f.From != nil {
		where.WriteString(" AND created_at >= ?")
		args = append(args, *f.From)
	}
	if f.To != nil {
		where.WriteString(" AND created_at < ?")
		args = append(args, *f.To)
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM transactions "+where.String(), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, store_id, code, shift_id, table_id, self_order_id, cashier_id, order_type, source,
		 payment_method, status, subtotal, discount, tax, service_charge, gateway_fee, total,
		 amount_received, change_amount, discount_approved_by, customer_note, created_at FROM transactions `+
			where.String()+" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]sqlcgen.Transaction, 0, f.Limit)
	for rows.Next() {
		var t sqlcgen.Transaction
		if err := rows.Scan(&t.ID, &t.StoreID, &t.Code, &t.ShiftID, &t.TableID, &t.SelfOrderID,
			&t.CashierID, &t.OrderType, &t.Source, &t.PaymentMethod, &t.Status, &t.Subtotal,
			&t.Discount, &t.Tax, &t.ServiceCharge, &t.GatewayFee, &t.Total, &t.AmountReceived, &t.ChangeAmount, &t.DiscountApprovedBy,
			&t.CustomerNote, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

// nullStr trims a string and returns a NULL sql.NullString when empty.
func nullStr(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
