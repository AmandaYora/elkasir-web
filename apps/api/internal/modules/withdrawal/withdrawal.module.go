// Package withdrawal wires the withdrawal module and exposes its HTTP handler.
package withdrawal

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	platformuserclient "github.com/elkasir/api/internal/modules/platformuser/contracts"
	salesclient "github.com/elkasir/api/internal/modules/transaction/contracts"
	"github.com/elkasir/api/internal/modules/withdrawal/application"
	withdrawalclient "github.com/elkasir/api/internal/modules/withdrawal/contracts"
	"github.com/elkasir/api/internal/modules/withdrawal/infrastructure"
	"github.com/elkasir/api/internal/modules/withdrawal/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/mail"
)

// Module is the assembled withdrawal module.
type Module struct {
	Handler *presentation.Handler
	// Client is the withdrawalclient.Client contract — consumed by the `platform` module for
	// the superadmin claim/complete flow + revenue reconciliation (Phase B5).
	Client withdrawalclient.Client
}

// New assembles the withdrawal module: repo → service → handler. Consumes the transaction
// module's sales contract (QRIS self-order revenue — §2.6's AvailableBalance basis), the
// platformuser contract + a mail.Sender (best-effort superadmin notification on submit, §2.10).
func New(pool *sql.DB, q *sqlcgen.Queries, auth authcontract.Authenticator, salesClient salesclient.Client, platformUsers platformuserclient.Client, mailer *mail.Sender, publicBaseURL string) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo, salesClient, platformUsers, mailer, publicBaseURL)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
		Client:  svc,
	}
}
