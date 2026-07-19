// Package adminuser wires the adminuser module (composition of its layered packages) and
// exposes its HTTP handler.
package adminuser

import (
	"database/sql"

	"github.com/elkasir/api/internal/modules/adminuser/application"
	adminuserclient "github.com/elkasir/api/internal/modules/adminuser/contracts"
	"github.com/elkasir/api/internal/modules/adminuser/infrastructure"
	"github.com/elkasir/api/internal/modules/adminuser/presentation"
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Module is the assembled adminuser module.
type Module struct {
	Handler *presentation.Handler
	// Client is the adminuserclient.Client contract — consumed by the `platform` module for the
	// superadmin cross-tenant admin-password-reset flow. This module's own /admin-users/* routes
	// stay self-service (store-scoped from the caller's own token); everything else stays private.
	Client adminuserclient.Client
}

// New assembles the adminuser module: repo → service → handler.
func New(pool *sql.DB, q *sqlcgen.Queries, auth authcontract.Authenticator) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
		Client:  svc,
	}
}
