// Package platformuser wires the platformuser module (superadmin account management). This
// module has NO HTTP handler and NO routes — `platform` owns the /platform/users/* routes and
// reaches this module only through platformuserclient.Client (same pattern as `payment`).
package platformuser

import (
	"database/sql"

	"github.com/elkasir/api/internal/modules/platformuser/application"
	platformuserclient "github.com/elkasir/api/internal/modules/platformuser/contracts"
	"github.com/elkasir/api/internal/modules/platformuser/infrastructure"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Module is the assembled platformuser module.
type Module struct {
	Client platformuserclient.Client
}

// New assembles the platformuser module: repo → service.
func New(pool *sql.DB, q *sqlcgen.Queries) *Module {
	repo := infrastructure.NewRepo(pool, q)
	svc := application.NewService(repo)
	return &Module{Client: svc}
}
