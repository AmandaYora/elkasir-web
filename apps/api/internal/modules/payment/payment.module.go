// Package payment wires the payment module (QRIS gateway over the Xendit adapter) and
// exposes ONLY its contract client. This module has NO HTTP handler and NO routes.
package payment

import (
	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/modules/payment/infrastructure"
	"github.com/elkasir/api/internal/platform/config"
	uow "github.com/elkasir/api/internal/platform/uow"
)

// Module is the assembled payment module.
type Module struct {
	Client paymentclient.Client
}

// New assembles the payment module: the tx-aware contract client over the Xendit adapter.
func New(cfg config.Xendit, uowMgr *uow.Manager) *Module {
	return &Module{
		Client: infrastructure.NewClient(cfg, uowMgr),
	}
}
