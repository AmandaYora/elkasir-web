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
	"log/slog"
	"time"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/modules/subscription/application"
	subscriptionclient "github.com/elkasir/api/internal/modules/subscription/contracts"
	"github.com/elkasir/api/internal/modules/subscription/infrastructure"
	"github.com/elkasir/api/internal/modules/subscription/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// reconcileInterval is how often StartReconciler polls ElProof for pending invoices (PLAN.md
// §11 Part C) — frequent enough to catch a lost webhook relay promptly, generous enough not to
// stress ElProof's per-app_id rate limit (60 req/min) even with many pending invoices at once.
const reconcileInterval = 2 * time.Minute

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

// StartReconciler launches the ElProof status-check poller (PLAN.md §11 Part C) as a background
// goroutine, ticking every reconcileInterval until ctx is cancelled. Call once from app.go, after
// this module is constructed, passing the process's root/shutdown context — same lifetime as the
// HTTP server itself. A single-process, in-memory ticker is enough here (same "self-hosted, no
// extra infra" philosophy already established for the payment module's rate limiter).
func (m *Module) StartReconciler(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(reconcileInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checked, resolved, err := m.svc.ReconcilePending(ctx)
				if err != nil {
					slog.Warn("subscription: reconciliation sweep gagal", "err", err)
					continue
				}
				if checked > 0 {
					slog.Info("subscription: reconciliation sweep", "checked", checked, "resolved", resolved)
				}
			}
		}
	}()
}
