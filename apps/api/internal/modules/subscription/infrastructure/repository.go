// Package infrastructure holds the subscription module's persistence (sqlc + database/sql).
// Owns subscription_plans, store_subscriptions, subscription_invoices EXCLUSIVELY — no other
// module reads or writes these tables (selforder keeps its own, separate `payments` ledger).
package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"github.com/elkasir/api/internal/modules/subscription/domain"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/id"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

func (r *Repo) ListActivePlans(ctx context.Context) ([]domain.Plan, error) {
	rows, err := r.q.ListActiveSubscriptionPlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Plan, 0, len(rows))
	for _, p := range rows {
		out = append(out, toPlan(p))
	}
	return out, nil
}

func (r *Repo) GetPlan(ctx context.Context, planID string) (domain.Plan, error) {
	p, err := r.q.GetSubscriptionPlan(ctx, planID)
	if err != nil {
		return domain.Plan{}, err
	}
	return toPlan(p), nil
}

// ListAllPlans returns every plan (including inactive) — the platform/superadmin view, as
// opposed to ListActivePlans which tenants use to pick a plan at checkout.
func (r *Repo) ListAllPlans(ctx context.Context) ([]domain.Plan, error) {
	rows, err := r.q.ListAllSubscriptionPlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Plan, 0, len(rows))
	for _, p := range rows {
		out = append(out, toPlan(p))
	}
	return out, nil
}

// CreatePlan persists a new plan (platform/superadmin only).
func (r *Repo) CreatePlan(ctx context.Context, planID, code, name string, price int64, periodDays int32, isActive bool) error {
	return r.q.CreateSubscriptionPlan(ctx, sqlcgen.CreateSubscriptionPlanParams{
		ID: planID, Code: code, Name: name, Price: price, PeriodDays: periodDays, IsActive: isActive,
	})
}

// UpdatePlan updates an existing plan's terms (platform/superadmin only). n=0 means not found.
func (r *Repo) UpdatePlan(ctx context.Context, planID, name string, price int64, periodDays int32, isActive bool) (int64, error) {
	return r.q.UpdateSubscriptionPlan(ctx, sqlcgen.UpdateSubscriptionPlanParams{
		Name: name, Price: price, PeriodDays: periodDays, IsActive: isActive, ID: planID,
	})
}

// PlatformRevenue implements subscriptionclient.Client — total revenue lintas semua tenant.
func (r *Repo) PlatformRevenue(ctx context.Context) (int64, error) {
	return r.q.SumPaidSubscriptionInvoices(ctx)
}

func (r *Repo) GetByStore(ctx context.Context, storeID string) (domain.Subscription, error) {
	s, err := r.q.GetStoreSubscription(ctx, storeID)
	if err != nil {
		return domain.Subscription{}, err
	}
	return toSubscription(s), nil
}

// CreateInvoice persists a new pending invoice and returns the resulting read model. Returns
// domain.ErrInvoiceAlreadyPending (wrapped) if the store already has an unresolved invoice open —
// enforced at the DB level by migration 000025's generated-column unique index, the race-free
// backstop behind application.Service's app-level pre-check (see Checkout).
func (r *Repo) CreateInvoice(ctx context.Context, invID, storeID, planID, provider, providerRef string, amount int64) (domain.Invoice, error) {
	err := r.q.CreateSubscriptionInvoice(ctx, sqlcgen.CreateSubscriptionInvoiceParams{
		ID: invID, StoreID: storeID, PlanID: planID, Amount: amount,
		Status:      sqlcgen.SubscriptionInvoicesStatusPending,
		Provider:    sqlcgen.SubscriptionInvoicesProvider(provider),
		ProviderRef: nullStr(providerRef),
		// StoreIDShadow: a plain copy of storeID, NOT a real second piece of data — see migration
		// 000025's doc comment for why the generated pending-lock column can't read store_id
		// directly (InnoDB forbids it once store_id is the child side of an ON DELETE CASCADE FK).
		StoreIDShadow: nullStr(storeID),
	})
	if db.IsDuplicate(err) {
		return domain.Invoice{}, domain.ErrInvoiceAlreadyPending
	}
	if err != nil {
		return domain.Invoice{}, err
	}
	return domain.Invoice{
		ID: invID, StoreID: storeID, PlanID: planID, Amount: amount,
		Status: string(sqlcgen.SubscriptionInvoicesStatusPending), Provider: provider, ProviderRef: providerRef,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// ListPendingElProof returns up to `limit` ElProof-sourced invoices still pending, oldest first
// — backs the reconciliation poller (PLAN.md §11 Part C), needed now that webhook delivery for
// subscription billing depends on a real cross-server, best-effort, single-attempt relay from
// ElProof (unlike the old in-process dispatch, which never needed a fallback). limit bounds each
// tick to a fixed-size batch — mirrors ElProof's own reconcileBatchLimit on its sweep — so a
// large backlog can't turn one tick into an unbounded burst of outbound requests.
func (r *Repo) ListPendingElProof(ctx context.Context, limit int32) ([]domain.Invoice, error) {
	rows, err := r.q.ListPendingElProofInvoices(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Invoice, 0, len(rows))
	for _, row := range rows {
		out = append(out, toInvoice(row))
	}
	return out, nil
}

// GetPendingInvoice returns the store's most recent unresolved invoice, if any — backs the
// checkout double-submit guard (a store should never have two live charges open for the same
// billing intent). Returns sql.ErrNoRows (unwrapped, per this repo's existing convention — see
// GetByStore) when there is none.
func (r *Repo) GetPendingInvoice(ctx context.Context, storeID string) (domain.Invoice, error) {
	row, err := r.q.GetPendingSubscriptionInvoiceByStore(ctx, storeID)
	if err != nil {
		return domain.Invoice{}, err
	}
	return toInvoice(row), nil
}

// SetInvoiceProviderRef fills in the gateway's own charge reference AFTER a successful charge
// (Checkout creates the invoice before calling the gateway — see service.go — so this field
// isn't known yet at CreateInvoice time). Informational only for ElProof invoices: reconciliation
// keys off the invoice's own ID as orderRef, never providerRef (see payment/infrastructure/
// elproof.go's checkStatus doc) — a failure here is never fatal to checkout.
func (r *Repo) SetInvoiceProviderRef(ctx context.Context, invoiceID, providerRef string) error {
	return r.q.SetSubscriptionInvoiceProviderRef(ctx, sqlcgen.SetSubscriptionInvoiceProviderRefParams{
		ID: invoiceID, ProviderRef: nullStr(providerRef),
	})
}

func (r *Repo) ListInvoices(ctx context.Context, storeID string, limit, offset int32) ([]domain.Invoice, int64, error) {
	rows, err := r.q.ListSubscriptionInvoices(ctx, sqlcgen.ListSubscriptionInvoicesParams{StoreID: storeID, Limit: limit, Offset: offset})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.CountSubscriptionInvoices(ctx, storeID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Invoice, 0, len(rows))
	for _, row := range rows {
		out = append(out, toInvoice(row))
	}
	return out, total, nil
}

// MarkInvoiceTerminal closes out an invoice ElProof reports as genuinely done WITHOUT payment
// (expired/failed/refund) — status must be one of subscription_invoices' own enum values
// ('expired' | 'failed'); guarded by status='pending' at the SQL level (see the query) so this
// can never downgrade an invoice a concurrent webhook/reconciler tick already marked 'paid'.
func (r *Repo) MarkInvoiceTerminal(ctx context.Context, invoiceID, status string) error {
	_, err := r.q.MarkSubscriptionInvoiceTerminal(ctx, sqlcgen.MarkSubscriptionInvoiceTerminalParams{
		Status: sqlcgen.SubscriptionInvoicesStatus(status), ID: invoiceID,
	})
	return err
}

// MarkInvoicePaidAndExtend is the ONE atomic operation triggered by payment confirmation: mark
// the invoice paid (guarded by status='pending' — idempotent against duplicate callbacks) and
// extend the store's subscription period. Both writes touch ONLY this module's own tables, so
// a plain DB transaction is enough — no cross-module Unit-of-Work needed. No-op (not an error)
// if the invoice is unknown or already processed.
func (r *Repo) MarkInvoicePaidAndExtend(ctx context.Context, invoiceID string, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := r.q.WithTx(tx)

	inv, err := q.GetSubscriptionInvoiceByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	if inv.Status != sqlcgen.SubscriptionInvoicesStatusPending {
		return nil // sudah diproses (idempoten)
	}
	plan, err := q.GetSubscriptionPlan(ctx, inv.PlanID)
	if err != nil {
		return err
	}

	start := now
	if sub, serr := q.GetStoreSubscription(ctx, inv.StoreID); serr == nil &&
		sub.CurrentPeriodEnd.Valid && sub.CurrentPeriodEnd.Time.After(now) {
		start = sub.CurrentPeriodEnd.Time // perpanjang dari akhir periode berjalan, bukan dari sekarang
	}
	end := start.Add(time.Duration(plan.PeriodDays) * 24 * time.Hour)

	n, err := q.MarkSubscriptionInvoicePaid(ctx, sqlcgen.MarkSubscriptionInvoicePaidParams{
		PeriodStart: sql.NullTime{Time: start, Valid: true},
		PeriodEnd:   sql.NullTime{Time: end, Valid: true},
		ID:          invoiceID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return nil // race — invoice sudah tak pending lagi di antara Get & Update
	}
	if err := q.UpsertStoreSubscriptionPeriod(ctx, sqlcgen.UpsertStoreSubscriptionPeriodParams{
		ID: id.New(), StoreID: inv.StoreID, PlanID: inv.PlanID,
		CurrentPeriodStart: sql.NullTime{Time: start, Valid: true},
		CurrentPeriodEnd:   sql.NullTime{Time: end, Valid: true},
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func toPlan(p sqlcgen.SubscriptionPlan) domain.Plan {
	return domain.Plan{
		ID: p.ID, Code: p.Code, Name: p.Name, Price: p.Price, PeriodDays: p.PeriodDays,
		IsActive: p.IsActive, RenewalOnly: p.RenewalOnly,
	}
}

func toSubscription(s sqlcgen.StoreSubscription) domain.Subscription {
	out := domain.Subscription{StoreID: s.StoreID, PlanID: s.PlanID, Status: string(s.Status)}
	if s.CurrentPeriodStart.Valid {
		out.CurrentPeriodStart = &s.CurrentPeriodStart.Time
	}
	if s.CurrentPeriodEnd.Valid {
		out.CurrentPeriodEnd = &s.CurrentPeriodEnd.Time
	}
	return out
}

func toInvoice(i sqlcgen.SubscriptionInvoice) domain.Invoice {
	out := domain.Invoice{
		ID: i.ID, StoreID: i.StoreID, PlanID: i.PlanID, Amount: i.Amount,
		Status: string(i.Status), Provider: string(i.Provider), ProviderRef: i.ProviderRef.String,
		CreatedAt: i.CreatedAt,
	}
	if i.PeriodStart.Valid {
		out.PeriodStart = &i.PeriodStart.Time
	}
	if i.PeriodEnd.Valid {
		out.PeriodEnd = &i.PeriodEnd.Time
	}
	return out
}
