// Package infrastructure holds the withdrawal module's persistence (sqlc + database/sql).
package infrastructure

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/elkasir/api/internal/modules/withdrawal/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

// List returns the store's withdrawals (paginated).
func (r *Repo) List(ctx context.Context, f domain.ListFilter) ([]domain.Withdrawal, error) {
	rows, err := r.q.ListWithdrawals(ctx, sqlcgen.ListWithdrawalsParams{StoreID: f.StoreID, Limit: f.Limit, Offset: f.Offset})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Withdrawal, 0, len(rows))
	for _, w := range rows {
		out = append(out, domain.Withdrawal{
			ID: w.ID, Amount: w.Amount, Bank: w.Bank, Account: w.Account, Holder: w.Holder,
			Status: string(w.Status), Reference: w.Reference.String, RequestedBy: w.RequestedBy.String,
			CreatedAt: w.CreatedAt,
		})
	}
	return out, nil
}

// Count returns the total number of withdrawals for the store.
func (r *Repo) Count(ctx context.Context, storeID string) (int64, error) {
	return r.q.CountWithdrawals(ctx, storeID)
}

// Create persists a new pending withdrawal and returns the resulting read model.
func (r *Repo) Create(ctx context.Context, storeID, id, requestedBy string, in domain.Input) (domain.Withdrawal, error) {
	bank := strings.TrimSpace(in.Bank)
	account := strings.TrimSpace(in.Account)
	holder := strings.TrimSpace(in.Holder)
	rb := nullStr(requestedBy)
	err := r.q.CreateWithdrawal(ctx, sqlcgen.CreateWithdrawalParams{
		ID: id, StoreID: storeID, Amount: in.Amount,
		Bank: bank, Account: account, Holder: holder,
		Status: sqlcgen.WithdrawalsStatusPending, RequestedBy: rb,
	})
	if err != nil {
		return domain.Withdrawal{}, err
	}
	return domain.Withdrawal{
		ID: id, Amount: in.Amount, Bank: bank, Account: account, Holder: holder,
		Status: string(sqlcgen.WithdrawalsStatusPending), RequestedBy: rb.String,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func nullStr(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
