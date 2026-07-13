// Package domain holds the subscription module's entities, value objects, and rules — tenant
// (store) billing to the elkasir platform. This is a DIFFERENT business domain from selforder
// (customer paying the store): here the store itself is the payer and elkasir is the payee.
// It reuses the same QRIS gateway (paymentclient.Client) but never shares a row or table with
// selforder's self_orders/payments.
package domain

import (
	"time"
)

// Plan is a subscription plan — reference/catalog data, managed by the platform (superadmin).
// RenewalOnly plans (e.g. the "Premium Contributor" legacy-backfill plan) can only ever be
// renewed by a subscriber already on them — never switched into from another plan, nor switched
// away from once assigned. Enforced in application.Service.Checkout, not editable via the
// platform plan CRUD form (see db/queries/subscriptions.sql).
type Plan struct {
	ID          string
	Code        string
	Name        string
	Price       int64
	PeriodDays  int32
	IsActive    bool
	RenewalOnly bool
}

// Subscription is a store's current subscription state (one row per store).
type Subscription struct {
	StoreID            string
	PlanID             string
	Status             string
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
}

// Invoice is one billing attempt for a store's subscription — this module's OWN gateway
// ledger, analogous to selforder's `payments` table but never shared with it.
type Invoice struct {
	ID          string
	StoreID     string
	PlanID      string
	Amount      int64
	Status      string
	Provider    string
	ProviderRef string
	PeriodStart *time.Time
	PeriodEnd   *time.Time
	CreatedAt   time.Time
}

