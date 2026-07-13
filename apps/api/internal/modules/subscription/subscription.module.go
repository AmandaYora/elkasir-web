// Package subscription wires the subscription module — tenant (store) billing to the elkasir
// platform. It reuses the SAME QRIS gateway as selforder (paymentclient.Client) but owns its
// own tables end to end (subscription_plans, store_subscriptions, subscription_invoices) — no
// row or table is shared with selforder's self_orders/payments. It exposes NO contract of its
// own (nothing else consumes it); the composition root reaches ApplyWebhookEvent directly on
// Module (registered as a paymentclient.WebhookConsumer, §9.1.5), the same pattern already used
// by selforder.
package subscription

import (
	"context"
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/modules/subscription/application"
	subscriptionclient "github.com/elkasir/api/internal/modules/subscription/contracts"
	"github.com/elkasir/api/internal/modules/subscription/infrastructure"
	"github.com/elkasir/api/internal/modules/subscription/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Module is the assembled subscription module.
type Module struct {
	Handler *presentation.Handler
	// Client is the subscriptionclient.Client contract — consumed by the `platform` module
	// for the superadmin cross-tenant revenue view. This is the module's FIRST contract;
	// everything else about subscription stays private (contract-ownership rule).
	Client subscriptionclient.Client
	svc    *application.Service
}

// New assembles the subscription module: repo → service → handler.
func New(pool *sql.DB, q *sqlcgen.Queries, auth authcontract.Authenticator, paymentClient paymentclient.Client) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo, paymentClient)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
		Client:  svc,
		svc:     svc,
	}
}

// ApplyWebhookEvent applies an already-verified/parsed/idempotency-checked gateway event to
// this module's own invoice + subscription period. Registered as a paymentclient.WebhookConsumer
// under paymentclient.AppSubscribe in app.go (§9.1.5) — dispatch is registry-driven now, not a
// "sub_"-prefix check.
func (m *Module) ApplyWebhookEvent(ctx context.Context, ev paymentclient.WebhookEvent) error {
	return m.svc.ApplyWebhookEvent(ctx, ev)
}
