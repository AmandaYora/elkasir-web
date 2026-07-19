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
	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/modules/platform"
	"github.com/elkasir/api/internal/modules/platformuser"
	"github.com/elkasir/api/internal/modules/product"
	"github.com/elkasir/api/internal/modules/report"
	"github.com/elkasir/api/internal/modules/selforder"
	"github.com/elkasir/api/internal/modules/settings"
	"github.com/elkasir/api/internal/modules/shift"
	"github.com/elkasir/api/internal/modules/staff"
	"github.com/elkasir/api/internal/modules/subscription"
	"github.com/elkasir/api/internal/modules/table"
	"github.com/elkasir/api/internal/modules/transaction"
	"github.com/elkasir/api/internal/modules/withdrawal"
	"github.com/elkasir/api/internal/platform/config"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpserver"
	"github.com/elkasir/api/internal/platform/mail"
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
	a.routes(ctx)
	return a, nil
}

// routes mounts health (root) + all business modules under /api/v1 + the embedded SPA
// (catch-all at root, registered last). ctx is the process's root/shutdown context (from
// signal.NotifyContext in cmd/api/main.go) — passed through so background loops started here
// (e.g. subscription's ElProof reconciler) share the server's own lifetime.
func (a *App) routes(ctx context.Context) {
	// Health/liveness at ROOT — for infra probes & container healthchecks.
	httpserver.RegisterHealth(a.Router, a.Pool)

	// Core auth module — provides the Authenticator other modules consume.
	authMod := auth.New(a.Queries, a.Cfg.JWT.Secret, a.Cfg.JWT.AccessTTL, a.Cfg.JWT.RefreshTTL)
	mw := authMod.Middleware

	// Staff is a provider too: it exposes the supervisor-PIN contract consumed by the shift &
	// transaction orchestrators (over-threshold approval), so it's assembled before them.
	staffMod := staff.New(a.Pool, a.Queries, mw)

	// Provider modules (expose contracts consumed by orchestrators), tx-aware via UoW.
	settingsMod := settings.New(a.Pool, a.Queries, a.UoW, mw)
	productMod := product.New(a.Pool, a.Queries, a.UoW, mw)
	shiftMod := shift.New(a.Pool, a.Queries, a.UoW, mw, staffMod.Client)
	tableMod := table.New(a.Pool, a.Queries, a.UoW, mw)
	paymentMod := payment.New(a.Cfg.Payment, a.UoW, a.Cfg.ConfigEncryptionKey, mw)
	txMod := transaction.New(a.Pool, a.Queries, a.UoW, mw, productMod.Client, shiftMod.Client, settingsMod.Client, staffMod.Client)

	// Leaf / consumer modules.
	categoryMod := category.New(a.Pool, a.Queries, mw)
	adminMod := adminuser.New(a.Pool, a.Queries, mw)
	// platformuser is contracts-only (no routes of its own — see Phase B3/B5); independent of
	// everything else, constructed here just so withdrawalMod can consume its contract below.
	platformUserMod := platformuser.New(a.Pool, a.Queries)
	mailer := mail.New(mail.Config{
		Host: a.Cfg.SMTP.Host, Port: a.Cfg.SMTP.Port, Username: a.Cfg.SMTP.Username,
		Password: a.Cfg.SMTP.Password, FromEmail: a.Cfg.SMTP.FromEmail, FromName: a.Cfg.SMTP.FromName,
	})
	withdrawalMod := withdrawal.New(a.Pool, a.Queries, mw, txMod.SalesClient, platformUserMod.Client, mailer, a.Cfg.PublicBaseURL)
	reportMod := report.New(a.Pool, a.Queries, mw)
	cashMod := cashmovement.New(a.Pool, a.Queries, mw, shiftMod.Client)
	selfMod := selforder.New(a.UoW, mw, productMod.Client, txMod.SalesClient, shiftMod.Client, tableMod.Client, paymentMod.Client, settingsMod.Client)

	// Tenant (store) billing to the platform itself — a SEPARATE business domain from selfMod
	// (customer paying the store). Reuses the SAME gateway (paymentMod.Client) but owns 100%
	// of its own tables; see knowledge/MODULE_MAP.md.
	subMod := subscription.New(a.Pool, a.Queries, mw, paymentMod.Client)
	// Wired here (not as an auth.New constructor param) to avoid a construction-order cycle —
	// subscription.New itself needs authMod's Middleware. See PLAN.md §1a/§3 (Phase B1.5).
	authMod.SetSubscriptionClient(subMod.Client)
	// ElProof status-check reconciler (PLAN.md §11 Part C) — subscription billing now depends on
	// a real cross-server webhook relay from ElProof (best-effort, single attempt), unlike the
	// old in-process dispatch which never needed a polling fallback.
	subMod.StartReconciler(ctx)

	// Register the two known internal webhook consumers (PLAN.md §9.1.4/§9.1.5, Part 2) —
	// replaces the old "sub_"-prefix sniffing that used to live in internal/app/webhook.go.
	// This is the ONE place allowed to know both consumers exist (same "composition root is the
	// only place aware of both" principle the old dispatcher followed).
	paymentMod.RegisterConsumer(paymentclient.AppSelfOrder, selfMod)
	paymentMod.RegisterConsumer(paymentclient.AppSubscribe, subMod)

	// Superadmin surface — tenant lifecycle + cross-tenant revenue. The ONE module allowed to
	// read across tenants, and only via subMod/txMod's own contracts (never a direct table read).
	platformMod := platform.New(a.Pool, a.Queries, mw, subMod.Client, txMod.SalesClient, withdrawalMod.Client, platformUserMod.Client, paymentMod.Client)

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
		settingsMod.Handler.Routes(r)
		subMod.Handler.Routes(r)
		platformMod.Handler.Routes(r)

		// ONE webhook endpoint shared by selforder + subscription — Tripay/Midtrans only
		// support a single callback URL per merchant account. Owned by the payment module
		// itself now (§9.1.5, Part 2) — dispatch is registry-driven (see RegisterConsumer
		// above), not a hardcoded two-consumer branch.
		paymentMod.Handler.Routes(r)
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
