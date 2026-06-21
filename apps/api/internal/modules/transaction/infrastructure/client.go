// Contract implementation for salesclient.Client. Tx-aware via uow: every insert goes
// through uow.Q(ctx) so it joins the transaction opened by an orchestrator (atomic with
// stock decrease and the rest of the cashier sale).
package infrastructure

import (
	"context"
	"database/sql"
	"strings"
	"time"

	salesclient "github.com/elkasir/api/internal/modules/transaction/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/id"
	"github.com/elkasir/api/internal/platform/uow"
)

type salesAdapter struct{ uow *uow.Manager }

// NewSalesClient membuat implementasi salesclient.Client.
func NewSalesClient(m *uow.Manager) salesclient.Client { return &salesAdapter{uow: m} }

var _ salesclient.Client = (*salesAdapter)(nil)

func (a *salesAdapter) RecordSale(ctx context.Context, in salesclient.RecordSaleInput) (string, error) {
	q := a.uow.Q(ctx)

	txID := id.New()
	if err := q.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		ID: txID, StoreID: in.StoreID, Code: "TRX-" + strings.ToUpper(txID[len(txID)-8:]),
		ShiftID: nullStr(in.ShiftID), TableID: nullStr(in.TableID),
		SelfOrderID: nullStr(in.SelfOrderID), CashierID: nullStr(in.CashierID),
		OrderType: orderType(in.OrderType), Source: txSource(in.Source), PaymentMethod: paymentMethod(in.PaymentMethod),
		Status:   sqlcgen.TransactionsStatusCompleted,
		Subtotal: in.Subtotal, Discount: in.Discount, Tax: in.Tax,
		ServiceCharge: in.ServiceCharge, GatewayFee: in.GatewayFee, Total: in.Total,
		AmountReceived: in.AmountReceived, ChangeAmount: in.Change,
		DiscountApprovedBy: nullStr(in.DiscountApprovedBy), CustomerNote: nullStr(in.CustomerNote),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return "", err
	}

	for _, it := range in.Items {
		if err := q.CreateTransactionItem(ctx, sqlcgen.CreateTransactionItemParams{
			ID: id.New(), TransactionID: txID,
			ProductID:   sql.NullString{String: it.ProductID, Valid: it.ProductID != ""},
			ProductName: it.ProductName, Category: it.Category,
			Price: it.Price, Quantity: it.Quantity, LineTotal: it.LineTotal,
			Note: sql.NullString{String: it.Note, Valid: it.Note != ""},
		}); err != nil {
			return "", err
		}
	}

	if in.IdempotencyKey != "" {
		if err := q.CreateIdempotencyKey(ctx, sqlcgen.CreateIdempotencyKeyParams{
			ID: id.New(), StoreID: in.StoreID, IdempotencyKey: in.IdempotencyKey, RequestHash: in.RequestHash,
			ResponseStatus: sql.NullInt32{Int32: 201, Valid: true},
			ResponseBody:   sql.NullString{String: txID, Valid: true},
		}); err != nil {
			return "", err
		}
	}

	return txID, nil
}

// orderType memetakan string order type ke enum sqlc.
func orderType(s string) sqlcgen.TransactionsOrderType {
	if s == "dineIn" {
		return sqlcgen.TransactionsOrderTypeDineIn
	}
	return sqlcgen.TransactionsOrderTypeTakeaway
}

// paymentMethod memetakan string metode bayar ke enum sqlc.
func paymentMethod(s string) sqlcgen.TransactionsPaymentMethod {
	if s == "qris" {
		return sqlcgen.TransactionsPaymentMethodQris
	}
	return sqlcgen.TransactionsPaymentMethodCash
}

// txSource memetakan string source ke enum sqlc.
func txSource(s string) sqlcgen.TransactionsSource {
	if s == "self_order" {
		return sqlcgen.TransactionsSourceSelfOrder
	}
	return sqlcgen.TransactionsSourceCashier
}
