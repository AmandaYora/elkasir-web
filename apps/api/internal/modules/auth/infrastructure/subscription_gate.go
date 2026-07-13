// Subscription-gated access (PLAN.md §2.15, Phase B1.5). auth depends on subscriptionclient,
// but NOT as a constructor param: subscription.New itself needs auth's Middleware to protect
// its own routes, so a constructor param would be a circular dependency in app.go. Instead,
// SetSubscriptionClient is called once in app.go right after subscription.New(...) returns —
// see PLAN.md §1a/§3's dependency notes.
package infrastructure

import (
	"context"
	"net/http"
	"strings"
	"time"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	subscriptionclient "github.com/elkasir/api/internal/modules/subscription/contracts"
	"github.com/elkasir/api/internal/platform/httpx"
)

const (
	staffPackageInactiveMessage = "Toko ini belum memiliki paket langganan aktif. Hubungi pemilik toko untuk memperbarui langganan."
	adminPackageInactiveMessage = "Paket langganan toko Anda tidak aktif. Perbarui paket langganan untuk melanjutkan."
)

// subscriptionGateAllowedPrefixes is the set of routes an ActorAdmin may still reach while
// their tenant's package is inactive (§2.15) — account/billing routes only, so the owner can
// always get to Langganan to fix it. PLAN.md §2.15 itself specifies this as a prefix
// ("GET/POST /subscription*"), not a closed list of exact paths — matched accordingly below.
//
// Prefix matching on r.URL.Path (not chi's RoutePattern()) — found via testing that
// chi collapses any sub-path under a Route()-mounted group into a wildcard leaf pattern
// (e.g. "/api/v1/subscription/plans" reports as "/api/v1/subscription/*" from
// RoutePattern()), so exact-pattern matching silently failed for every subscription route
// except the mount root itself. See PLAN.md §1a.
var subscriptionGateAllowedPrefixes = []string{
	"/api/v1/subscription", // covers /subscription, /subscription/plans, /checkout, /invoices
	"/api/v1/settings",
	"/api/v1/auth/logout",
	// Not in §2.15's literal list, but required for its own stated intent ("an admin can
	// always log in far enough to reach Langganan"): /auth/me is behind the same Authenticate
	// middleware as everything else, so without this the frontend's session restore
	// (GET /auth/me) would itself 402 and the app would treat that as an invalid session —
	// logging the owner out instead of showing the locked Langganan shell.
	"/api/v1/auth/me",
}

// SetSubscriptionClient wires the subscription contract into the middleware. See the package
// doc comment above for why this is a post-construction setter, not a constructor param.
func (mw *Middleware) SetSubscriptionClient(c subscriptionclient.Client) { mw.sub = c }

// hasActivePackage reports "punya paket aktif" per §2.15: a store_subscriptions row with
// status "active" AND currentPeriodEnd >= now. Computed live on every call — no caching, same
// philosophy as the tenant-suspension check.
func (mw *Middleware) hasActivePackage(ctx context.Context, storeID string) (bool, error) {
	sub, err := mw.sub.Current(ctx, storeID)
	if err != nil {
		return false, err
	}
	if sub.Status != "active" || sub.CurrentPeriodEnd == nil {
		return false, nil
	}
	return !sub.CurrentPeriodEnd.Before(time.Now()), nil
}

// allowlistedRoute reports whether the current request's path falls under one of the
// subscription-gate allowed prefixes (§2.15's ActorAdmin partial-block exception).
func allowlistedRoute(r *http.Request) bool {
	path := strings.TrimRight(r.URL.Path, "/")
	for _, prefix := range subscriptionGateAllowedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// checkSubscriptionGate runs the §2.15 gate for an already-suspension-cleared Admin/Staff
// principal. Returns a non-nil error (already written to w via httpx.Error by the caller) when
// the request must stop. Skipped entirely if the subscription client hasn't been wired yet
// (construction window in app.go, before SetSubscriptionClient runs) — see §1a.
func (mw *Middleware) checkSubscriptionGate(w http.ResponseWriter, r *http.Request, p authcontract.Principal) (stop bool) {
	if mw.sub == nil {
		return false
	}
	active, err := mw.hasActivePackage(r.Context(), p.StoreID)
	if err != nil {
		httpx.Error(w, err)
		return true
	}
	if active {
		return false
	}
	if p.Actor == authcontract.ActorStaff {
		httpx.Error(w, httpx.Forbidden(staffPackageInactiveMessage))
		return true
	}
	// ActorAdmin: partial block — allow only the subscription/account allowlist through.
	if allowlistedRoute(r) {
		return false
	}
	httpx.Error(w, httpx.PaymentRequired(adminPackageInactiveMessage))
	return true
}
