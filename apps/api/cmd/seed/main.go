// Command seed mengisi data awal MINIMAL (1 toko + setting default + 1 akun admin).
// Idempoten — aman dijalankan berulang. Logika inti ada di internal/platform/bootstrap
// sehingga dipakai bersama subcommand `api seed`.
package main

import (
	"context"
	"log"
	"time"

	"github.com/elkasir/api/internal/platform/bootstrap"
	"github.com/elkasir/api/internal/platform/config"
	"github.com/elkasir/api/internal/platform/db"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("seed: %v", err)
	}
}

func run() error {
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

	log.Printf("seed selesai: admin login %q / %q", bootstrap.AdminEmail, bootstrap.AdminPassword)
	log.Printf("seed selesai: platform superadmin login %q / %q", bootstrap.PlatformEmail, bootstrap.PlatformPassword)
	return nil
}
