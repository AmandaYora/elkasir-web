// Package product wires the product module and exposes its HTTP handler + contract client.
package product

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/product/application"
	productclient "github.com/elkasir/api/internal/modules/product/contracts"
	"github.com/elkasir/api/internal/modules/product/infrastructure"
	"github.com/elkasir/api/internal/modules/product/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/uow"
)

// Module is the assembled product module.
type Module struct {
	Handler *presentation.Handler
	Client  productclient.Client
}

// New assembles the product module: repo → service → handler, plus the tx-aware contract
// client that other modules consume.
func New(pool *sql.DB, q *sqlcgen.Queries, uowMgr *uow.Manager, auth authcontract.Authenticator) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
		Client:  infrastructure.NewClient(uowMgr),
	}
}
