// Package withdrawalclient is the PUBLIC contract of the withdrawal module — consumed by the
// `platform` module (superadmin claim/complete flow + revenue reconciliation, PLAN.md §2.6/§2.7).
package withdrawalclient

import (
	"context"
	"time"
)

// Withdrawal is the platform (superadmin)-facing read model of a withdrawal request — a
// superset of the tenant-facing DTO (this module's own application.DTO), since the superadmin
// view needs StoreID + the full claim/complete audit trail (§2.8).
type Withdrawal struct {
	ID             string     `json:"id"`
	StoreID        string     `json:"storeId"`
	Amount         int64      `json:"amount"`
	Bank           string     `json:"bank"`
	Account        string     `json:"account"`
	Holder         string     `json:"holder"`
	Status         string     `json:"status"`
	RequestedBy    string     `json:"requestedBy,omitempty"`
	ProcessedBy    string     `json:"processedBy,omitempty"`
	ClaimedAt      *time.Time `json:"claimedAt,omitempty"`
	ProcessedAt    *time.Time `json:"processedAt,omitempty"`
	RejectedReason string     `json:"rejectedReason,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// TenantBalance pairs a store with its AvailableBalance (§2.6) — Revenue Tenant page.
type TenantBalance struct {
	StoreID string `json:"storeId"`
	Balance int64  `json:"balance"`
}

// ListFilter paginates ListAll (cross-tenant, any status — Riwayat Penarikan page).
type ListFilter struct {
	Limit  int32
	Offset int32
}

// Client is the contract published by the withdrawal module.
type Client interface {
	// AvailableBalance is the reconciliation-accurate figure (§2.6) — displayed everywhere
	// (Ringkasan, Revenue Tenant, the tenant's own Withdrawals page).
	AvailableBalance(ctx context.Context, storeID string) (int64, error)
	// AvailableBalanceByTenant is the same basis, all tenants at once.
	AvailableBalanceByTenant(ctx context.Context) ([]TenantBalance, error)
	// TotalSuccessfulWithdrawals is cross-tenant — feeds GET /platform/revenue.
	TotalSuccessfulWithdrawals(ctx context.Context) (int64, error)
	// ListActive returns pending+processing requests, cross-tenant (Penarikan page).
	ListActive(ctx context.Context) ([]Withdrawal, error)
	// ListAll returns any-status requests, cross-tenant, paginated (Riwayat Penarikan page).
	ListAll(ctx context.Context, filter ListFilter) ([]Withdrawal, int64, error)
	// Claim moves pending -> processing (§2.7). Runs the claimable check (§2.6) and the
	// tenant-suspension check (§2.14). actorID is the claiming superadmin's platform_users.id.
	Claim(ctx context.Context, id, actorID string) error
	// MarkSuccess moves processing -> success (§2.7) — actorID must be who claimed it.
	MarkSuccess(ctx context.Context, id, actorID string) error
	// MarkRejected moves pending|processing -> failed (§2.7) — any active superadmin.
	MarkRejected(ctx context.Context, id, actorID, reason string) error
}
