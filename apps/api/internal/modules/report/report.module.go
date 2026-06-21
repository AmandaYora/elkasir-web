// Package report wires the report module (read-only analytics) and exposes its HTTP
// handler. The report module has NO contract and performs NO writes.
package report

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/report/application"
	"github.com/elkasir/api/internal/modules/report/infrastructure"
	"github.com/elkasir/api/internal/modules/report/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Module is the assembled report module.
type Module struct {
	Handler *presentation.Handler
}

// New assembles the report module: repo → service → handler.
func New(pool *sql.DB, q *sqlcgen.Queries, auth authcontract.Authenticator) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
	}
}
