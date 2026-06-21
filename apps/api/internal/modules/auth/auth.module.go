// Package auth wires the auth module (composition of its layered packages) and exposes
// the HTTP handler plus the Authenticator other modules depend on.
package auth

import (
	"time"

	"github.com/elkasir/api/internal/modules/auth/application"
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/auth/infrastructure"
	"github.com/elkasir/api/internal/modules/auth/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Module is the assembled auth module.
type Module struct {
	Handler    *presentation.Handler
	Middleware authcontract.Authenticator
}

// New assembles the auth module: token manager → service → middleware → handler.
func New(q *sqlcgen.Queries, secret string, accessTTL, refreshTTL time.Duration) *Module {
	mgr := infrastructure.NewManager(secret, accessTTL, refreshTTL)
	svc := application.NewService(q, mgr)
	mw := infrastructure.NewMiddleware(mgr)
	return &Module{
		Handler:    presentation.NewHandler(svc, mw),
		Middleware: mw,
	}
}
