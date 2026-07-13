// Package payment wires the payment module (QRIS/VA gateway — Tripay/Midtrans, one active
// wallet, §9.1.1) and exposes its contract client + the ONE webhook route registered with the
// gateway. Superadmin-facing config/registry ROUTES live in `platform` (§9.1.10) — this module
// only exposes the contract methods `platform` calls, plus this one webhook endpoint.
package payment

import (
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/modules/payment/infrastructure"
	"github.com/elkasir/api/internal/modules/payment/presentation"
	"github.com/elkasir/api/internal/platform/config"
	uow "github.com/elkasir/api/internal/platform/uow"
)

// Module is the assembled payment module.
type Module struct {
	Client  paymentclient.Client
	Handler *presentation.Handler
}

// New assembles the payment module: the tx-aware contract client over the active gateway, the
// webhook handler, and (Part 3, §10.2) the external payment API routes gated by `auth`
// (ActorApp). encryptionKey is cfg-adjacent but passed separately (not part of config.Payment)
// since it protects DB-stored credentials, not the gateway call itself.
func New(cfg config.Payment, uowMgr *uow.Manager, encryptionKey string, auth authcontract.Authenticator) *Module {
	client := infrastructure.NewClient(cfg, uowMgr, encryptionKey)
	return &Module{
		Client:  client,
		Handler: presentation.NewHandler(client, auth),
	}
}

// RegisterConsumer wires an internal consumer (self-order, subscription — §9.1.5) into the
// webhook dispatcher. Called once per consumer from the composition root (app.go), after every
// module is constructed. Panics if Client doesn't implement Dispatcher — a programming error,
// not a runtime condition (the concrete apiClient always implements both).
func (m *Module) RegisterConsumer(appID string, consumer paymentclient.WebhookConsumer) {
	m.Client.(paymentclient.Dispatcher).RegisterConsumer(appID, consumer)
}
