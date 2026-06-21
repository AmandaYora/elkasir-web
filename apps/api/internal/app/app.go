// Package app is the composition root: it wires config, DB, modules, and the router.
package app

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/elkasir/api/internal/modules/adminuser"
	"github.com/elkasir/api/internal/modules/auth"
	"github.com/elkasir/api/internal/modules/cashmovement"
	"github.com/elkasir/api/internal/modules/category"
	"github.com/elkasir/api/internal/modules/media"
	"github.com/elkasir/api/internal/modules/payment"
	"github.com/elkasir/api/internal/modules/product"
	"github.com/elkasir/api/internal/modules/report"
	"github.com/elkasir/api/internal/modules/selforder"
	"github.com/elkasir/api/internal/modules/shift"
	"github.com/elkasir/api/internal/modules/staff"
	"github.com/elkasir/api/internal/modules/table"
	"github.com/elkasir/api/internal/modules/transaction"
	"github.com/elkasir/api/internal/modules/withdrawal"
	"github.com/elkasir/api/internal/platform/config"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpserver"
	"github.com/elkasir/api/internal/platform/storage"
	"github.com/elkasir/api/internal/platform/uow"
	"github.com/elkasir/api/internal/webui"
	"github.com/go-chi/chi/v5"
)

type App struct {
	Cfg     config.Config
	Pool    *sql.DB
	Queries *sqlcgen.Queries
	UoW     *uow.Manager // Unit of Work — injected into modules with atomic cross-module flows
	Router  *chi.Mux
}

// New opens the DB and assembles the full router with every module.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	pool, err := db.Open(ctx, cfg.DB.DSN)
	if err != nil {
		return nil, err
	}

	a := &App{
		Cfg:     cfg,
		Pool:    pool,
		Queries: sqlcgen.New(pool),
		UoW:     uow.New(pool),
		Router:  httpserver.NewRouter(cfg),
	}
	a.routes()
	return a, nil
}

// routes mounts health (root) + all business modules under /api/v1 + the embedded SPA
// (catch-all at root, registered last).
func (a *App) routes() {
	// Health/liveness at ROOT — for infra probes & container healthchecks.
	httpserver.RegisterHealth(a.Router, a.Pool)

	// Core auth module — provides the Authenticator other modules consume.
	authMod := auth.New(a.Queries, a.Cfg.JWT.Secret, a.Cfg.JWT.AccessTTL, a.Cfg.JWT.RefreshTTL)
	mw := authMod.Middleware

	// Provider modules (expose contracts consumed by orchestrators), tx-aware via UoW.
	productMod := product.New(a.Pool, a.Queries, a.UoW, mw)
	shiftMod := shift.New(a.Pool, a.Queries, a.UoW, mw)
	tableMod := table.New(a.Pool, a.Queries, a.UoW, mw)
	paymentMod := payment.New(a.Cfg.Xendit, a.UoW)
	txMod := transaction.New(a.Pool, a.Queries, a.UoW, mw, productMod.Client, shiftMod.Client)

	// Leaf / consumer modules.
	categoryMod := category.New(a.Pool, a.Queries, mw)
	staffMod := staff.New(a.Pool, a.Queries, mw)
	adminMod := adminuser.New(a.Pool, a.Queries, mw)
	withdrawalMod := withdrawal.New(a.Pool, a.Queries, mw)
	reportMod := report.New(a.Pool, a.Queries, mw)
	cashMod := cashmovement.New(a.Pool, a.Queries, mw, shiftMod.Client)
	selfMod := selforder.New(a.UoW, mw, productMod.Client, txMod.SalesClient, shiftMod.Client, tableMod.Client, paymentMod.Client)

	// Object storage (S3-compatible) — opsional. Bila kredensial belum diisi, klien
	// dibiarkan nil dan endpoint upload mengembalikan error yang jelas.
	mediaMod := media.New(a.newStorage(), mw)

	// All business APIs are versioned under /api/v1 so the SPA (served at root) and the API
	// never collide. The web client uses base URL "/api/v1".
	a.Router.Route("/api/v1", func(r chi.Router) {
		authMod.Handler.Routes(r)

		productMod.Handler.Routes(r)
		categoryMod.Handler.Routes(r)
		tableMod.Handler.Routes(r)
		staffMod.Handler.Routes(r)
		adminMod.Handler.Routes(r)
		mediaMod.Handler.Routes(r)

		shiftMod.Handler.Routes(r)
		txMod.Handler.Routes(r)
		cashMod.Handler.Routes(r)
		withdrawalMod.Handler.Routes(r)

		reportMod.Handler.Routes(r)
		selfMod.Handler.Routes(r)
	})

	// Static SPA (embedded in the binary). Catch-all at root, registered LAST — serves
	// assets & falls back to index.html for client routes. 1 binary = web + API.
	a.Router.Handle("/*", webui.Handler())
}

// newStorage builds the object-storage client when configured. A misconfiguration is
// non-fatal: it logs a warning and returns nil so the app still boots (uploads off).
func (a *App) newStorage() *storage.Client {
	if !a.Cfg.Storage.Enabled() {
		slog.Warn("object storage disabled: OBJSTORE_* not set — image uploads will fail")
		return nil
	}
	sc, err := storage.New(a.Cfg.Storage)
	if err != nil {
		slog.Error("object storage init failed; uploads disabled", "err", err)
		return nil
	}
	slog.Info("object storage enabled", "bucket", a.Cfg.Storage.Bucket, "endpoint", a.Cfg.Storage.Endpoint)
	return sc
}

// Close releases resources (DB pool).
func (a *App) Close() error {
	if a.Pool != nil {
		return a.Pool.Close()
	}
	return nil
}
