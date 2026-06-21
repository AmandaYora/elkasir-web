// Package settings wires the settings module: owns the settings table, exposes the admin
// HTTP handler (GET/PATCH /settings) AND the read contract (settingsclient) for other modules.
package settings

import (
	"database/sql"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	settingsclient "github.com/elkasir/api/internal/modules/settings/contracts"
	"github.com/elkasir/api/internal/modules/settings/application"
	"github.com/elkasir/api/internal/modules/settings/infrastructure"
	"github.com/elkasir/api/internal/modules/settings/presentation"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	uow "github.com/elkasir/api/internal/platform/uow"
)

// Module is the assembled settings module.
type Module struct {
	Handler *presentation.Handler
	Client  settingsclient.Client
}

// New assembles the settings module: repo → service → handler, plus the tx-aware read client.
func New(pool *sql.DB, q *sqlcgen.Queries, uowMgr *uow.Manager, auth authcontract.Authenticator) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo)
	return &Module{
		Handler: presentation.NewHandler(svc, auth),
		Client:  infrastructure.NewClient(uowMgr),
	}
}
