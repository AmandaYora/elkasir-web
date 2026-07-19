// Package application holds the platform module's use cases — tenant lifecycle management +
// cross-tenant revenue, for the superadmin (ActorPlatform) only. It orchestrates tenant
// creation via bootstrap.ProvisionTenant and reads revenue aggregates through the subscription
// and transaction contracts — it never touches their tables directly.
package application

import (
	"context"
	"database/sql"
	"sort"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/modules/platform/domain"
	"github.com/elkasir/api/internal/modules/platform/infrastructure"
	platformuserclient "github.com/elkasir/api/internal/modules/platformuser/contracts"
	subscriptionclient "github.com/elkasir/api/internal/modules/subscription/contracts"
	salesclient "github.com/elkasir/api/internal/modules/transaction/contracts"
	withdrawalclient "github.com/elkasir/api/internal/modules/withdrawal/contracts"
	"github.com/elkasir/api/internal/platform/bootstrap"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/httpx"
)

type Service struct {
	repo          *infrastructure.Repo
	pool          *sql.DB
	subscription  subscriptionclient.Client
	sales         salesclient.Client
	withdrawals   withdrawalclient.Client
	platformUsers platformuserclient.Client
	payments      paymentclient.Client
}

func NewService(repo *infrastructure.Repo, pool *sql.DB, subscriptionClient subscriptionclient.Client, salesClient salesclient.Client, withdrawalClient withdrawalclient.Client, platformUserClient platformuserclient.Client, paymentClient paymentclient.Client) *Service {
	return &Service{
		repo: repo, pool: pool, subscription: subscriptionClient, sales: salesClient,
		withdrawals: withdrawalClient, platformUsers: platformUserClient, payments: paymentClient,
	}
}

// WithdrawalView is a withdrawal enriched with the tenant's name and (if claimed) the
// claimant's name — the Penarikan/Riwayat Penarikan pages need both, and neither `withdrawal`
// nor `platformuser` can resolve them (they don't own `stores`/each other's tables).
type WithdrawalView struct {
	withdrawalclient.Withdrawal
	TenantName   string `json:"tenantName"`
	ClaimantName string `json:"claimantName,omitempty"`
}

func (s *Service) ListTenants(ctx context.Context) ([]domain.Tenant, error) {
	return s.repo.List(ctx)
}

// CreateTenant onboards a brand-new tenant: store + default settings + owner admin account,
// all in one transaction (bootstrap.ProvisionTenant) — this is currently the ONLY way a new
// tenant comes into existence (no self-registration flow).
func (s *Service) CreateTenant(ctx context.Context, in domain.CreateTenantInput) (domain.Tenant, error) {
	if err := in.Validate(); err != nil {
		return domain.Tenant{}, err
	}
	storeID, err := bootstrap.ProvisionTenant(ctx, s.pool, bootstrap.ProvisionTenantInput{
		StoreName: in.StoreName, StoreSlug: in.StoreSlug,
		OwnerName: in.OwnerName, OwnerEmail: in.OwnerEmail, OwnerPassword: in.OwnerPassword,
	})
	if db.IsDuplicate(err) {
		return domain.Tenant{}, httpx.Conflict("Slug toko atau email pemilik sudah dipakai.")
	}
	if err != nil {
		return domain.Tenant{}, err
	}
	return s.repo.Get(ctx, storeID)
}

// SetTenantStatus activates/suspends a tenant. Suspension is a lifecycle flag only in this
// pass — it does not yet block login/API access; enforcing it is a follow-up (would need every
// admin/staff auth check to also read stores.status).
func (s *Service) SetTenantStatus(ctx context.Context, storeID, status string) (domain.Tenant, error) {
	if status != "active" && status != "suspended" {
		return domain.Tenant{}, httpx.Validation("Status harus 'active' atau 'suspended'.")
	}
	n, err := s.repo.SetStatus(ctx, storeID, status)
	if err != nil {
		return domain.Tenant{}, err
	}
	if n == 0 {
		return domain.Tenant{}, httpx.NotFound("Tenant tidak ditemukan.")
	}
	return s.repo.Get(ctx, storeID)
}

// Revenue returns the cross-tenant reconciliation dashboard (§2.5): subscription revenue
// (platform's own, from `subscription`) + tenants' unwithdrawn QRIS balance (self-order QRIS
// GMV from `transaction`, minus successful withdrawals from `withdrawal`) — together these
// should equal the real Tripay/Midtrans gateway balance. Neither cross-tenant call is
// store-scoped — each is the ONE deliberate exception in its owning module's contract.
func (s *Service) Revenue(ctx context.Context) (domain.RevenueSummary, error) {
	subRev, err := s.subscription.PlatformRevenue(ctx)
	if err != nil {
		return domain.RevenueSummary{}, err
	}
	qrisRev, err := s.sales.PlatformSelfOrderQrisRevenue(ctx)
	if err != nil {
		return domain.RevenueSummary{}, err
	}
	successWithdrawals, err := s.withdrawals.TotalSuccessfulWithdrawals(ctx)
	if err != nil {
		return domain.RevenueSummary{}, err
	}
	tenantAvailable := qrisRev - successWithdrawals
	return domain.RevenueSummary{
		SubscriptionRevenue: subRev, TenantAvailableBalance: tenantAvailable,
		TotalMonitored: subRev + tenantAvailable,
	}, nil
}

// TenantsRevenue joins this module's own tenant list (stores — name/slug) with
// withdrawalclient.AvailableBalanceByTenant() by storeID, in Go (not SQL — different modules'
// tables). Sorted by balance descending (Revenue Tenant page, §2.6, read-only).
func (s *Service) TenantsRevenue(ctx context.Context) ([]domain.TenantRevenue, error) {
	tenants, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	balances, err := s.withdrawals.AvailableBalanceByTenant(ctx)
	if err != nil {
		return nil, err
	}
	balanceByStore := make(map[string]int64, len(balances))
	for _, b := range balances {
		balanceByStore[b.StoreID] = b.Balance
	}
	out := make([]domain.TenantRevenue, 0, len(tenants))
	for _, t := range tenants {
		out = append(out, domain.TenantRevenue{StoreID: t.ID, Name: t.Name, Slug: t.Slug, Balance: balanceByStore[t.ID]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Balance > out[j].Balance })
	return out, nil
}

// ── Withdrawal processing passthrough (withdrawal module owns the flow; this module only
// enriches with tenant/claimant names for display, §2.7) ─────────────────────────────────────

func (s *Service) ListActiveWithdrawals(ctx context.Context) ([]WithdrawalView, error) {
	rows, err := s.withdrawals.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	return s.enrichWithdrawals(ctx, rows)
}

func (s *Service) ListWithdrawalHistory(ctx context.Context, limit, offset int32) ([]WithdrawalView, int64, error) {
	rows, total, err := s.withdrawals.ListAll(ctx, withdrawalclient.ListFilter{Limit: limit, Offset: offset})
	if err != nil {
		return nil, 0, err
	}
	views, err := s.enrichWithdrawals(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return views, total, nil
}

func (s *Service) ClaimWithdrawal(ctx context.Context, id, actorID string) error {
	return s.withdrawals.Claim(ctx, id, actorID)
}

func (s *Service) CompleteWithdrawal(ctx context.Context, id, actorID string) error {
	return s.withdrawals.MarkSuccess(ctx, id, actorID)
}

func (s *Service) RejectWithdrawal(ctx context.Context, id, actorID, reason string) error {
	return s.withdrawals.MarkRejected(ctx, id, actorID, reason)
}

func (s *Service) enrichWithdrawals(ctx context.Context, rows []withdrawalclient.Withdrawal) ([]WithdrawalView, error) {
	tenants, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	tenantNames := make(map[string]string, len(tenants))
	for _, t := range tenants {
		tenantNames[t.ID] = t.Name
	}
	users, err := s.platformUsers.List(ctx)
	if err != nil {
		return nil, err
	}
	userNames := make(map[string]string, len(users))
	for _, u := range users {
		userNames[u.ID] = u.Name
	}
	out := make([]WithdrawalView, 0, len(rows))
	for _, w := range rows {
		view := WithdrawalView{Withdrawal: w, TenantName: tenantNames[w.StoreID]}
		if w.ProcessedBy != "" {
			view.ClaimantName = userNames[w.ProcessedBy]
		}
		out = append(out, view)
	}
	return out, nil
}

// ── Platform-user management passthrough (platformuser owns the table; §2.9) ─────────────────

func (s *Service) ListPlatformUsers(ctx context.Context) ([]platformuserclient.PlatformUser, error) {
	return s.platformUsers.List(ctx)
}

func (s *Service) CreatePlatformUser(ctx context.Context, in platformuserclient.CreateInput) (platformuserclient.PlatformUser, error) {
	return s.platformUsers.Create(ctx, in)
}

func (s *Service) SetPlatformUserStatus(ctx context.Context, actingUserID, targetID, status string) error {
	return s.platformUsers.SetStatus(ctx, actingUserID, targetID, status)
}

func (s *Service) ResetPlatformUserPassword(ctx context.Context, id, password string) error {
	return s.platformUsers.ResetPassword(ctx, id, password)
}

// ── Plan management passthrough (subscription_plans is owned by `subscription`; this module
// only orchestrates the superadmin-facing surface via its contract — see subscriptionclient.Client) ──

func (s *Service) ListPlans(ctx context.Context) ([]subscriptionclient.Plan, error) {
	return s.subscription.ListAllPlans(ctx)
}

func (s *Service) CreatePlan(ctx context.Context, in subscriptionclient.PlanInput) (subscriptionclient.Plan, error) {
	return s.subscription.CreatePlan(ctx, in)
}

func (s *Service) UpdatePlan(ctx context.Context, planID string, in subscriptionclient.PlanInput) (subscriptionclient.Plan, error) {
	return s.subscription.UpdatePlan(ctx, planID, in)
}

// ── Payment gateway config + app registry passthrough (PLAN.md §9.1.10 — payment owns no
// routes of its own for this; platform exposes the superadmin surface via paymentclient.Client,
// same pattern as every other passthrough above) ─────────────────────────────────────────────

func (s *Service) GetPaymentConfig(ctx context.Context) (paymentclient.GatewayConfig, error) {
	return s.payments.GetConfig(ctx)
}

func (s *Service) UpdatePaymentConfig(ctx context.Context, in paymentclient.UpdateGatewayConfigInput) (paymentclient.GatewayConfig, error) {
	return s.payments.UpdateConfig(ctx, in)
}
