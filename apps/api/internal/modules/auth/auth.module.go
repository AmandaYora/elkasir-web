// Package auth wires the auth module (composition of its layered packages) and exposes
// the HTTP handler plus the Authenticator other modules depend on.
package auth

import (
	"time"

	"github.com/elkasir/api/internal/modules/auth/application"
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/auth/infrastructure"
	"github.com/elkasir/api/internal/modules/auth/presentation"
	subscriptionclient "github.com/elkasir/api/internal/modules/subscription/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Module is the assembled auth module.
type Module struct {
	Handler    *presentation.Handler
	Middleware authcontract.Authenticator
	mw         *infrastructure.Middleware // concrete ref for SetSubscriptionClient (Phase B1.5)
	svc        *application.Service       // concrete ref for the same reason (login-time gate)
}

// New assembles the auth module: token manager → service → middleware → handler.
func New(q *sqlcgen.Queries, secret string, accessTTL, refreshTTL time.Duration) *Module {
	mgr := infrastructure.NewManager(secret, accessTTL, refreshTTL)
	svc := application.NewService(q, mgr)
	mw := infrastructure.NewMiddleware(mgr, q)
	return &Module{
		Handler:    presentation.NewHandler(svc, mw),
		Middleware: mw,
		mw:         mw,
		svc:        svc,
	}
}

// SetSubscriptionClient wires the subscription contract into both the middleware's
// package-inactive gate and the service's staff-login gate (§2.15). Must be called once in
// app.go, right after subscription.New(...) returns and before the server accepts requests —
// see PLAN.md §1a/§3 for why this can't be a constructor param (subscription.New itself needs
// this module's Middleware).
func (m *Module) SetSubscriptionClient(c subscriptionclient.Client) {
	m.mw.SetSubscriptionClient(c)
	m.svc.SetSubscriptionClient(c)
}
