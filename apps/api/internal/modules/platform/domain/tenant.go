// Package domain holds the platform module's entities and rules — tenant (store) lifecycle
// management + cross-tenant revenue view, operated by the superadmin (ActorPlatform). This is
// the ONE module in the whole app whose normal operation is deliberately cross-tenant; every
// other module stays strictly scoped by store_id.
package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/elkasir/api/internal/platform/httpx"
)

// Tenant is the platform's view of a store (read model).
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateTenantInput provisions a brand-new tenant + its first owner account in one go — there
// is no other way to onboard a tenant (no self-registration flow exists).
type CreateTenantInput struct {
	StoreName     string `json:"storeName"`
	StoreSlug     string `json:"storeSlug"`
	OwnerName     string `json:"ownerName"`
	OwnerEmail    string `json:"ownerEmail"`
	OwnerPassword string `json:"ownerPassword"`
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Validate enforces tenant-creation rules. StoreSlug becomes part of the self-order QR URL
// (/order/<slug>/<tableCode>) — this is also the fix for the table-code collision bug found
// in the multi-tenancy audit (table codes are only unique per-store).
func (in CreateTenantInput) Validate() error {
	if strings.TrimSpace(in.StoreName) == "" {
		return httpx.Validation("Nama toko wajib diisi.")
	}
	if !slugPattern.MatchString(in.StoreSlug) {
		return httpx.Validation("Slug hanya boleh huruf kecil, angka, dan tanda hubung (mis. warkop-budi).")
	}
	if strings.TrimSpace(in.OwnerName) == "" || strings.TrimSpace(in.OwnerEmail) == "" {
		return httpx.Validation("Nama dan email pemilik toko wajib diisi.")
	}
	if len(in.OwnerPassword) < 6 {
		return httpx.Validation("Password pemilik toko minimal 6 karakter.")
	}
	return nil
}

// RevenueSummary is the superadmin's reconciliation dashboard (§2.5): subscription revenue
// (platform's own) + tenants' unwithdrawn QRIS balance, which together should equal the real
// Tripay/Midtrans gateway balance (manual sanity check, not automated). Cash self-order never
// touches the gateway and is deliberately excluded — this is NOT a sales report.
type RevenueSummary struct {
	SubscriptionRevenue    int64 `json:"subscriptionRevenue"`
	TenantAvailableBalance int64 `json:"tenantAvailableBalance"`
	TotalMonitored         int64 `json:"totalMonitored"`
}

// TenantRevenue pairs a tenant with its AvailableBalance (§2.6) — Revenue Tenant page, read-only.
type TenantRevenue struct {
	StoreID string `json:"storeId"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Balance int64  `json:"balance"`
}
