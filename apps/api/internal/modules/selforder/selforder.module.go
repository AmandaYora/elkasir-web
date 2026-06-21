// Package selforder wires the selforder module — the customer self-order ORCHESTRATOR.
// It exposes only an HTTP handler (public customer endpoints + admin/staff endpoints) and
// provides NO contract of its own. It CONSUMES five module contracts (product, transaction
// sales, shift, table, payment) and orchestrates an atomic checkout via the unit-of-work.
package selforder

import (
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	productclient "github.com/elkasir/api/internal/modules/product/contracts"
	"github.com/elkasir/api/internal/modules/selforder/application"
	"github.com/elkasir/api/internal/modules/selforder/infrastructure"
	"github.com/elkasir/api/internal/modules/selforder/presentation"
	settingsclient "github.com/elkasir/api/internal/modules/settings/contracts"
	shiftclient "github.com/elkasir/api/internal/modules/shift/contracts"
	tableclient "github.com/elkasir/api/internal/modules/table/contracts"
	salesclient "github.com/elkasir/api/internal/modules/transaction/contracts"
	"github.com/elkasir/api/internal/platform/uow"
)

// Module is the assembled selforder module.
type Module struct {
	Handler *presentation.Handler
}

// New assembles the selforder module: repo (built from the uow manager) → service →
// handler. It consumes the product, sales, shift, table, and payment contracts to
// orchestrate place-order and an atomic checkout/fulfilment via the unit-of-work.
func New(
	uowMgr *uow.Manager,
	auth authcontract.Authenticator,
	productClient productclient.Client,
	salesClient salesclient.Client,
	shiftClient shiftclient.Client,
	tableClient tableclient.Client,
	paymentClient paymentclient.Client,
	settingsClient settingsclient.Client,
) *Module {
	repo := infrastructure.NewRepo(uowMgr)
	svc := application.NewService(repo, productClient, salesClient, shiftClient, tableClient, paymentClient, settingsClient, uowMgr)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
	}
}
