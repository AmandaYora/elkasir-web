package infrastructure

import (
	"net/http"
	"strings"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/platform/httpx"
)

// Middleware wraps token validation. It implements authcontract.Authenticator so other
// modules depend only on the contract, not on this concrete type.
type Middleware struct{ mgr *Manager }

func NewMiddleware(mgr *Manager) *Middleware { return &Middleware{mgr: mgr} }

var _ authcontract.Authenticator = (*Middleware)(nil)

// Authenticate validates the Bearer token and puts the principal in the context (401 on failure).
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
		next.ServeHTTP(w, r.WithContext(authcontract.WithPrincipal(r.Context(), p)))
	})
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
