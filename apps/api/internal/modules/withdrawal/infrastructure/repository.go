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
	return toDomainList(rows), nil
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
		ID: id, StoreID: storeID, Amount: in.Amount, Bank: bank, Account: account, Holder: holder,
		Status: string(sqlcgen.WithdrawalsStatusPending), RequestedBy: rb.String,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// Get returns a single withdrawal by id (any tenant — used by the superadmin claim/complete
// flow, §2.7, which is deliberately cross-tenant).
func (r *Repo) Get(ctx context.Context, id string) (domain.Withdrawal, error) {
	w, err := r.q.GetWithdrawal(ctx, id)
	if err != nil {
		return domain.Withdrawal{}, err
	}
	return toDomain(w), nil
}

// StoreSuspended reads stores.status directly — same narrow, read-only shared-kernel exception
// as auth's tenant-suspension check (§2.14), used by Claim.
func (r *Repo) StoreSuspended(ctx context.Context, storeID string) (bool, error) {
	status, err := r.q.GetStoreStatus(ctx, storeID)
	if err != nil {
		return false, err
	}
	return status == sqlcgen.StoresStatusSuspended, nil
}

// ListActive returns pending+processing withdrawals, cross-tenant (Penarikan page, §2.7).
func (r *Repo) ListActive(ctx context.Context) ([]domain.Withdrawal, error) {
	rows, err := r.q.ListActiveWithdrawals(ctx)
	if err != nil {
		return nil, err
	}
	return toDomainList(rows), nil
}

// ListAll returns any-status withdrawals, cross-tenant, paginated (Riwayat Penarikan page).
func (r *Repo) ListAll(ctx context.Context, limit, offset int32) ([]domain.Withdrawal, int64, error) {
	rows, err := r.q.ListAllWithdrawals(ctx, sqlcgen.ListAllWithdrawalsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.CountAllWithdrawals(ctx)
	if err != nil {
		return nil, 0, err
	}
	return toDomainList(rows), total, nil
}

// SumSuccessfulByStore is the §2.6 AvailableBalance basis for one tenant.
func (r *Repo) SumSuccessfulByStore(ctx context.Context, storeID string) (int64, error) {
	return r.q.SumSuccessfulWithdrawalsByStore(ctx, storeID)
}

// SumSuccessfulAll is cross-tenant — feeds GET /platform/revenue.
func (r *Repo) SumSuccessfulAll(ctx context.Context) (int64, error) {
	return r.q.SumSuccessfulWithdrawals(ctx)
}

// SumSuccessfulGroupedByStore is the §2.6 AvailableBalance basis, all tenants at once.
func (r *Repo) SumSuccessfulGroupedByStore(ctx context.Context) (map[string]int64, error) {
	rows, err := r.q.SumSuccessfulWithdrawalsGroupedByStore(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.StoreID] = row.Total
	}
	return out, nil
}

// SumProcessingByStore is the §2.6 claimable-check basis (narrower than AvailableBalance).
func (r *Repo) SumProcessingByStore(ctx context.Context, storeID string) (int64, error) {
	return r.q.SumProcessingWithdrawalsByStore(ctx, storeID)
}

// Claim atomically moves pending -> processing (§2.7). Returns rows-affected — 0 means the
// request was no longer pending (already claimed/rejected by someone else).
func (r *Repo) Claim(ctx context.Context, id, actorID string) (int64, error) {
	return r.q.ClaimWithdrawal(ctx, sqlcgen.ClaimWithdrawalParams{
		ProcessedBy: nullStr(actorID),
		ClaimedAt:   sql.NullTime{Time: time.Now().UTC(), Valid: true},
		ID:          id,
	})
}

// MarkSuccess atomically moves processing -> success (§2.7), only if actorID matches the
// claimant. Returns rows-affected — 0 means not processing anymore, or actorID isn't the claimant.
func (r *Repo) MarkSuccess(ctx context.Context, id, actorID string) (int64, error) {
	return r.q.MarkWithdrawalSuccess(ctx, sqlcgen.MarkWithdrawalSuccessParams{
		ProcessedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true},
		ID:          id,
		ProcessedBy: nullStr(actorID),
	})
}

// MarkRejected atomically moves pending|processing -> failed (§2.7). Returns rows-affected —
// 0 means the request had already reached a terminal state.
func (r *Repo) MarkRejected(ctx context.Context, id, actorID, reason string) (int64, error) {
	return r.q.MarkWithdrawalRejected(ctx, sqlcgen.MarkWithdrawalRejectedParams{
		ProcessedBy:    nullStr(actorID),
		ProcessedAt:    sql.NullTime{Time: time.Now().UTC(), Valid: true},
		RejectedReason: nullStr(reason),
		ID:             id,
	})
}

func toDomain(w sqlcgen.Withdrawal) domain.Withdrawal {
	out := domain.Withdrawal{
		ID: w.ID, StoreID: w.StoreID, Amount: w.Amount, Bank: w.Bank, Account: w.Account, Holder: w.Holder,
		Status: string(w.Status), Reference: w.Reference.String, RequestedBy: w.RequestedBy.String,
		ProcessedBy: w.ProcessedBy.String, RejectedReason: w.RejectedReason.String,
		CreatedAt: w.CreatedAt,
	}
	if w.ClaimedAt.Valid {
		out.ClaimedAt = &w.ClaimedAt.Time
	}
	if w.ProcessedAt.Valid {
		out.ProcessedAt = &w.ProcessedAt.Time
	}
	return out
}

func toDomainList(rows []sqlcgen.Withdrawal) []domain.Withdrawal {
	out := make([]domain.Withdrawal, 0, len(rows))
	for _, w := range rows {
		out = append(out, toDomain(w))
	}
	return out
}

func nullStr(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
