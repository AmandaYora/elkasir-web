// Package withdrawal wires the withdrawal module and exposes its HTTP handler.
package withdrawal

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/withdrawal/application"
	"github.com/elkasir/api/internal/modules/withdrawal/infrastructure"
	"github.com/elkasir/api/internal/modules/withdrawal/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Module is the assembled withdrawal module.
type Module struct {
	Handler *presentation.Handler
}

// New assembles the withdrawal module: repo → service → handler.
func New(pool *sql.DB, q *sqlcgen.Queries, auth authcontract.Authenticator) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
	}
}
