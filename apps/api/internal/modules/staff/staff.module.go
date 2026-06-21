// Package staff wires the staff module (composition of its layered packages) and exposes
// its HTTP handler.
package staff

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/staff/application"
	"github.com/elkasir/api/internal/modules/staff/infrastructure"
	"github.com/elkasir/api/internal/modules/staff/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Module is the assembled staff module.
type Module struct {
	Handler *presentation.Handler
}

// New assembles the staff module: repo → service → handler.
func New(pool *sql.DB, q *sqlcgen.Queries, auth authcontract.Authenticator) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
	}
}
