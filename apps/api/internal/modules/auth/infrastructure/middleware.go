package infrastructure

import (
	"context"
	"net/http"
	"strings"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	subscriptionclient "github.com/elkasir/api/internal/modules/subscription/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpx"
)

// Middleware wraps token validation. It implements authcontract.Authenticator so other
// modules depend only on the contract, not on this concrete type.
//
// Per PLAN.md §2.13/§1a, this is the first time this middleware performs any per-request I/O
// (previously pure JWT parse/verify) — a straight, uncached DB read on every authenticated
// request is an intentional, LOCKED design choice, not an oversight.
type Middleware struct {
	mgr *Manager
	q   *sqlcgen.Queries
	sub subscriptionclient.Client // set post-construction — see subscription_gate.go
}

func NewMiddleware(mgr *Manager, q *sqlcgen.Queries) *Middleware { return &Middleware{mgr: mgr, q: q} }

var _ authcontract.Authenticator = (*Middleware)(nil)

// Authenticate validates the Bearer token, enforces tenant suspension (§2.13), and puts the
// principal in the context. 401 on an invalid/expired token, 403 if the principal's tenant is
// suspended.
func (mw *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := bearerToken(r)
		if raw == "" {
			httpx.Error(w, httpx.Unauthorized("Autentikasi diperlukan."))
			return
		}
		p, err := mw.mgr.ParseAccess(raw)
		if err != nil {
			httpx.Error(w, httpx.Unauthorized("Token tidak valid atau kedaluwarsa."))
			return
		}
		if p.StoreID != "" {
			suspended, err := mw.tenantSuspended(r.Context(), p.StoreID)
			if err != nil {
				httpx.Error(w, err)
				return
			}
			if suspended {
				httpx.Error(w, httpx.Forbidden("Toko Anda sedang dinonaktifkan. Hubungi platform."))
				return
			}
			if mw.checkSubscriptionGate(w, r, p) {
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(authcontract.WithPrincipal(r.Context(), p)))
	})
}

// tenantSuspended reads stores.status directly — a narrow, read-only, precedented kind of
// shared-kernel access (same justification class as settings/platform's existing exceptions,
// PLAN.md §2.13/§2.14), not table ownership by auth. ActorPlatform principals have no StoreID
// and never reach this check (see the caller).
func (mw *Middleware) tenantSuspended(ctx context.Context, storeID string) (bool, error) {
	status, err := mw.q.GetStoreStatus(ctx, storeID)
	if err != nil {
		return false, err
	}
	return status == sqlcgen.StoresStatusSuspended, nil
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
