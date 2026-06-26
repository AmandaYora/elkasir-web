// Package shift wires the shift module and exposes its HTTP handler + contract client.
package shift

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/shift/application"
	shiftclient "github.com/elkasir/api/internal/modules/shift/contracts"
	"github.com/elkasir/api/internal/modules/shift/infrastructure"
	"github.com/elkasir/api/internal/modules/shift/presentation"
	staffclient "github.com/elkasir/api/internal/modules/staff/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/uow"
)

// Module is the assembled shift module.
type Module struct {
	Handler *presentation.Handler
	Client  shiftclient.Client
}

// New assembles the shift module: repo → service → handler, plus the tx-aware contract
// client that other modules consume. staffClient verifies the supervisor PIN when a cashier
// closes a shift with an over-tolerance cash variance.
func New(pool *sql.DB, q *sqlcgen.Queries, uowMgr *uow.Manager, auth authcontract.Authenticator, staffClient staffclient.Client) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo, staffClient)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
		Client:  infrastructure.NewClient(uowMgr),
	}
}
