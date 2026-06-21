// Package cashmovement wires the cashmovement module and exposes its HTTP handler.
// It CONSUMES the shift contract (shiftclient) and does NOT provide a contract.
package cashmovement

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/cashmovement/application"
	"github.com/elkasir/api/internal/modules/cashmovement/infrastructure"
	"github.com/elkasir/api/internal/modules/cashmovement/presentation"
	shiftclient "github.com/elkasir/api/internal/modules/shift/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Module is the assembled cashmovement module.
type Module struct {
	Handler *presentation.Handler
}

// New assembles the cashmovement module: repo → service → handler. The service consumes
// the shift contract to attribute movements to the current open shift.
func New(pool *sql.DB, q *sqlcgen.Queries, auth authcontract.Authenticator, shiftClient shiftclient.Client) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo, shiftClient)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
	}
}
