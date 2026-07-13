// Command api adalah titik masuk backend Elkasir (modular monolith). Satu binary,
// banyak peran lewat subcommand argv[1] — sehingga image distroless yang sama bisa
// serve, migrate, seed, dan healthcheck (lihat docs/DEPLOYMENT_PIPELINE.md §9):
//
//	api               serve (default) — load config → DB → router (chi) → serve
//	api serve         sama dengan default
//	api migrate up    terapkan migrasi go:embed
//	api migrate down [n]  mundur n langkah (default 1)
//	api seed          isi data awal minimal (idempoten)
//	api healthcheck   GET /readyz lokal → exit 0/1 (dipakai Docker HEALTHCHECK)
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/elkasir/api/internal/app"
	"github.com/elkasir/api/internal/platform/bootstrap"
	"github.com/elkasir/api/internal/platform/config"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/httpserver"
	"github.com/elkasir/api/internal/platform/migrator"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	args := os.Args[1:]
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}

	var err error
	switch cmd {
	case "", "serve":
		err = run()
	case "migrate":
		err = runMigrate(args[1:])
	case "seed":
		err = runSeed()
	case "healthcheck":
		err = runHealthcheck()
	default:
		err = fmt.Errorf("perintah tidak dikenal: %q (pakai: serve|migrate|seed|healthcheck)", cmd)
	}

	if err != nil {
		slog.Error("fatal", "cmd", cmd, "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = application.Close() }()

	srv := httpserver.NewHTTPServer(cfg.Addr, application.Router)

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("api_listening", "addr", cfg.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		slog.Info("shutdown_signal_received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	slog.Info("shutdown_complete")
	return nil
}

// runMigrate menjalankan `migrate up` (default) atau `migrate down [n]`.
func runMigrate(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	direction := "up"
	if len(args) > 0 {
		direction = args[0]
	}

	switch direction {
	case "up":
		slog.Info("migrate_up_start")
		if err := migrator.Up(cfg.DB.DSN); err != nil {
			return err
		}
		slog.Info("migrate_up_done")
		return nil
	case "down":
		n := 1
		if len(args) > 1 {
			v, convErr := strconv.Atoi(args[1])
			if convErr != nil {
				return fmt.Errorf("migrate down: jumlah langkah tidak valid: %q", args[1])
			}
			n = v
		}
		slog.Info("migrate_down_start", "steps", n)
		if err := migrator.Down(cfg.DB.DSN, n); err != nil {
			return err
		}
		slog.Info("migrate_down_done")
		return nil
	default:
		return fmt.Errorf("migrate: arah tidak dikenal: %q (up|down [n])", direction)
	}
}

// runSeed mengisi data awal minimal (idempoten).
func runSeed() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, cfg.DB.DSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := bootstrap.Seed(ctx, pool); err != nil {
		return err
	}
	slog.Info("seed_done", "admin_email", bootstrap.AdminEmail, "platform_email", bootstrap.PlatformEmail)
	return nil
}

// runHealthcheck memanggil /readyz lokal dan keluar 0 (siap) atau 1 (belum).
// Dipakai sebagai Docker HEALTHCHECK pada image distroless (tanpa shell/curl).
func runHealthcheck() error {
	addr := strings.TrimSpace(os.Getenv("API_ADDR"))
	if addr == "" {
		addr = ":8081"
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + addr + "/readyz")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: status %d", resp.StatusCode)
	}
	return nil
}
