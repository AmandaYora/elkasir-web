// Package transaction wires the transaction module (cross-module cashier sale
// orchestrator) and exposes its HTTP handler + the sales contract client other modules
// consume.
package transaction

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	productclient "github.com/elkasir/api/internal/modules/product/contracts"
	settingsclient "github.com/elkasir/api/internal/modules/settings/contracts"
	shiftclient "github.com/elkasir/api/internal/modules/shift/contracts"
	"github.com/elkasir/api/internal/modules/transaction/application"
	salesclient "github.com/elkasir/api/internal/modules/transaction/contracts"
	"github.com/elkasir/api/internal/modules/transaction/infrastructure"
	"github.com/elkasir/api/internal/modules/transaction/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/uow"
)

// Module is the assembled transaction module.
type Module struct {
	Handler     *presentation.Handler
	SalesClient salesclient.Client
}

// New assembles the transaction module: repo + tx-aware sales client → service → handler.
// It builds its own sales client internally and consumes the product & shift contracts to
// orchestrate an atomic cashier sale via the unit-of-work.
func New(pool *sql.DB, q *sqlcgen.Queries, uowMgr *uow.Manager, auth authcontract.Authenticator,
	productClient productclient.Client, shiftClient shiftclient.Client, settingsClient settingsclient.Client) *Module {
	repo := infrastructure.NewRepo(pool, q)
	salesClient := infrastructure.NewSalesClient(uowMgr)
	svc := application.NewService(repo, productClient, shiftClient, salesClient, settingsClient, uowMgr)
	return &Module{
		Handler:     presentation.NewHandler(svc, auth),
		SalesClient: salesClient,
	}
}
