// Package subscriptionclient is the PUBLIC contract of the subscription module. The only
// consumer today is the `platform` module (superadmin dashboard: revenue + plan catalog
// management) — this is why the contract is deliberately narrow: one cross-tenant revenue
// aggregate, plus plan-catalog CRUD. Nothing about any single tenant's billing detail is
// exposed here.
package subscriptionclient

import (
	"context"
	"time"
)

// Subscription is a store's current billing/period snapshot. Status "none" means the tenant
// has never checked out (not an error). Consumed by `auth`'s subscription-gate middleware
// (PLAN.md §2.15, narrow read-only contract access) and by this module's own tenant-facing
// GET /subscription handler.
type Subscription struct {
	Status             string     `json:"status"`
	PlanID             string     `json:"planId,omitempty"`
	// PlanName/PlanPrice/PlanPeriodDays/PlanRenewalOnly are resolved regardless of the plan's
	// is_active flag — a subscriber's plan may be a hidden one (e.g. the §2.15 backfill "Premium
	// Contributor" plan) that no longer appears in the tenant-facing active-plans list, but its
	// details must still display (and drive UI rules) on the Langganan page — the frontend can't
	// just look it up in that list. PlanRenewalOnly in particular is what the frontend uses to
	// explicitly hide "other plans"/upgrade options — not just as a side effect of the plan being
	// absent from the active list (that alone wouldn't hold if a future plan were both active AND
	// renewal_only).
	PlanName           string     `json:"planName,omitempty"`
	PlanPrice          int64      `json:"planPrice,omitempty"`
	PlanPeriodDays     int32      `json:"planPeriodDays,omitempty"`
	PlanRenewalOnly    bool       `json:"planRenewalOnly,omitempty"`
	CurrentPeriodStart *time.Time `json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   *time.Time `json:"currentPeriodEnd,omitempty"`
}

// Plan is a subscription plan (reference/catalog data). RenewalOnly plans (e.g. the
// "Premium Contributor" legacy-backfill plan) can only ever be renewed by a subscriber already
// on them — see domain.Plan's doc comment for the full rule.
type Plan struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Price       int64  `json:"price"`
	PeriodDays  int32  `json:"periodDays"`
	IsActive    bool   `json:"isActive"`
	RenewalOnly bool   `json:"renewalOnly"`
}

// PlanInput is the create/update payload for a plan. Code is only used on create — a plan's
// code is a stable identity, not editable afterwards.
type PlanInput struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Price      int64  `json:"price"`
	PeriodDays int32  `json:"periodDays"`
	IsActive   bool   `json:"isActive"`
}

// Client is the contract published by the subscription module.
type Client interface {
	// PlatformRevenue returns total revenue (rupiah) from ALL PAID invoices, ACROSS ALL
	// TENANTS — the one deliberate exception to this app's "always filter by store_id" rule,
	// reserved for the platform/superadmin revenue view.
	PlatformRevenue(ctx context.Context) (int64, error)
	// ListAllPlans returns every plan, including inactive ones (platform/superadmin view).
	ListAllPlans(ctx context.Context) ([]Plan, error)
	CreatePlan(ctx context.Context, in PlanInput) (Plan, error)
	UpdatePlan(ctx context.Context, planID string, in PlanInput) (Plan, error)
	// Current returns the given store's subscription snapshot — used by `auth`'s
	// subscription-gate middleware (§2.15) to compute "has active package"
	// (status == "active" && currentPeriodEnd >= now). Narrow, read-only.
	Current(ctx context.Context, storeID string) (Subscription, error)
}
