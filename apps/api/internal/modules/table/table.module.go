// Package table wires the table module and exposes its HTTP handler + contract client.
package table

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/table/application"
	tableclient "github.com/elkasir/api/internal/modules/table/contracts"
	"github.com/elkasir/api/internal/modules/table/infrastructure"
	"github.com/elkasir/api/internal/modules/table/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/uow"
)

// Module is the assembled table module.
type Module struct {
	Handler *presentation.Handler
	Client  tableclient.Client
}

// New assembles the table module: repo → service → handler, plus the tx-aware contract
// client that other modules consume.
func New(pool *sql.DB, q *sqlcgen.Queries, uowMgr *uow.Manager, auth authcontract.Authenticator) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
		Client:  infrastructure.NewClient(uowMgr),
	}
}
