// Package category wires the category module and exposes its HTTP handler.
package category

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/category/application"
	"github.com/elkasir/api/internal/modules/category/infrastructure"
	"github.com/elkasir/api/internal/modules/category/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Module is the assembled category module.
type Module struct {
	Handler *presentation.Handler
}

// New assembles the category module: repo → service → handler.
func New(pool *sql.DB, q *sqlcgen.Queries, auth authcontract.Authenticator) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
	}
}
