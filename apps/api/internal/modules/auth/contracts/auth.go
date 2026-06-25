// Package authcontract is the PUBLIC boundary of the auth module. Other modules import
// ONLY this package to protect routes and read the authenticated principal — never the
// auth application/infrastructure/presentation packages.
package authcontract

import (
	"context"
	"net/http"

	"github.com/elkasir/api/internal/platform/httpx"
)

// Actor distinguishes the two identity contexts (do not mix them).
type Actor string

const (
	ActorAdmin Actor = "admin"
	ActorStaff Actor = "staff"
)

// Principal is the authenticated identity carried in the request context.
type Principal struct {
	SubjectID string
	StoreID   string
	Actor     Actor
	Role      string
}

// IsSupervisorOrAdmin reports whether the principal is privileged enough to perform
// (or self-approve) supervisor-gated actions: any admin web user, or a staff supervisor.
// A plain cashier returns false and must obtain a supervisor's approval (PIN).
func (p Principal) IsSupervisorOrAdmin() bool {
	return p.Actor == ActorAdmin || (p.Actor == ActorStaff && p.Role == "supervisor")
}

// Authenticator validates the bearer token and injects the principal into the context.
// The concrete implementation lives in the auth module's infrastructure.
type Authenticator interface {
	Authenticate(next http.Handler) http.Handler
}

type ctxKey int

const principalKey ctxKey = iota

// WithPrincipal stores a principal in the context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// PrincipalFrom reads the principal from the context.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

// MustPrincipal returns the principal (handlers behind Authenticate always have one).
func MustPrincipal(ctx context.Context) Principal {
	p, _ := PrincipalFrom(ctx)
	return p
}

// RequireActor rejects (403) when the identity context does not match (admin vs staff).
func RequireActor(actor Actor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFrom(r.Context())
			if !ok || p.Actor != actor {
				httpx.Error(w, httpx.Forbidden("Akses tidak diizinkan untuk konteks ini."))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireStaffSupervisorOrAdmin gates supervisor-only POS surfaces: any admin web user is
// allowed (the web dashboard manages everything), but a staff (POS) principal must be a
// supervisor — a plain cashier is rejected (403). Use for cash movements, reports, and other
// "kasir = kasir saja, sisanya supervisor" endpoints.
func RequireStaffSupervisorOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFrom(r.Context())
		if !ok {
			httpx.Error(w, httpx.Unauthorized("Autentikasi diperlukan."))
			return
		}
		if !p.IsSupervisorOrAdmin() {
			httpx.Error(w, httpx.Forbidden("Hanya supervisor yang dapat mengakses fitur ini."))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole rejects (403) when the principal's role is not in the allowed set.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFrom(r.Context())
			if !ok {
				httpx.Error(w, httpx.Unauthorized("Autentikasi diperlukan."))
				return
			}
			if _, allow := allowed[p.Role]; !allow {
				httpx.Error(w, httpx.Forbidden("Role Anda tidak punya izin untuk aksi ini."))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
