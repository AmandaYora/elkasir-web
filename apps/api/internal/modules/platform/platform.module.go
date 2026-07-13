// Package platform wires the platform module — the superadmin (ActorPlatform) surface: tenant
// lifecycle (create/list/suspend), cross-tenant revenue, and subscription-plan catalog
// management. It is the ONE module in this app whose normal operation deliberately crosses
// tenant boundaries; it does so ONLY through the subscription and transaction contracts (never
// a direct table read/write into their tables), and it owns no table of its own — it just
// reads/writes the tenant-lifecycle columns on the shared-kernel `stores` table.
package platform

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/modules/platform/application"
	"github.com/elkasir/api/internal/modules/platform/infrastructure"
	"github.com/elkasir/api/internal/modules/platform/presentation"
	platformuserclient "github.com/elkasir/api/internal/modules/platformuser/contracts"
	subscriptionclient "github.com/elkasir/api/internal/modules/subscription/contracts"
	salesclient "github.com/elkasir/api/internal/modules/transaction/contracts"
	withdrawalclient "github.com/elkasir/api/internal/modules/withdrawal/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Module is the assembled platform module.
type Module struct {
	Handler *presentation.Handler
}

// New assembles the platform module: repo → service → handler. Consumes 5 contracts:
// subscription (revenue + plan catalog), transaction (self-order QRIS GMV), withdrawal
// (claim/complete flow + balance reconciliation, Phase B2), platformuser (superadmin account
// management, Phase B3), payment (gateway config + app registry, PLAN.md §9.1.10, Part 2).
func New(pool *sql.DB, q *sqlcgen.Queries, auth authcontract.Authenticator, subscriptionClient subscriptionclient.Client, salesClient salesclient.Client, withdrawalClient withdrawalclient.Client, platformUserClient platformuserclient.Client, paymentClient paymentclient.Client) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo, pool, subscriptionClient, salesClient, withdrawalClient, platformUserClient, paymentClient)
	return &Module{Handler: presentation.NewHandler(svc, auth)}
}
