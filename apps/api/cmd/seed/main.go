// Command seed mengisi data awal MINIMAL: 1 toko + setting default + 1 akun admin
// (email "admin", password "admin123"). Idempoten — aman dijalankan berulang.
package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/elkasir/api/internal/platform/config"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/id"
	"golang.org/x/crypto/bcrypt"
)

const (
	storeName = "Elkasir"
	storeType = "Restoran"
	storeTZ   = "Asia/Jakarta"

	adminName     = "Administrator"
	adminEmail    = "admin"
	adminPassword = "admin123"
	adminRole     = "owner"
	adminStatus   = "active"

	maxDiscountPercent    = 10
	maxOperationalExpense = 200000
	cashVarianceTolerance = 5000
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

	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	storeID, err := ensureStore(ctx, tx)
	if err != nil {
		return err
	}
	if err := upsertSettings(ctx, tx, storeID); err != nil {
		return err
	}
	if err := upsertAdmin(ctx, tx, storeID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("seed selesai: store=%s, admin login %q / %q", storeID, adminEmail, adminPassword)
	return nil
}

func ensureStore(ctx context.Context, tx *sql.Tx) (string, error) {
	var storeID string
	err := tx.QueryRowContext(ctx, "SELECT id FROM stores WHERE name = ? LIMIT 1", storeName).Scan(&storeID)
	if errors.Is(err, sql.ErrNoRows) {
		storeID = id.New()
		_, err = tx.ExecContext(ctx,
			"INSERT INTO stores (id, name, type, timezone) VALUES (?, ?, ?, ?)",
			storeID, storeName, storeType, storeTZ)
		return storeID, err
	}
	return storeID, err
}

func upsertSettings(ctx context.Context, tx *sql.Tx, storeID string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO settings (id, store_id, max_discount_percent, max_operational_expense, cash_variance_tolerance)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			max_discount_percent = VALUES(max_discount_percent),
			max_operational_expense = VALUES(max_operational_expense),
			cash_variance_tolerance = VALUES(cash_variance_tolerance)`,
		id.New(), storeID, maxDiscountPercent, maxOperationalExpense, cashVarianceTolerance)
	return err
}

func upsertAdmin(ctx context.Context, tx *sql.Tx, storeID string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO admin_users (id, store_id, name, email, password_hash, role, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			store_id = VALUES(store_id), name = VALUES(name),
			password_hash = VALUES(password_hash), role = VALUES(role), status = VALUES(status)`,
		id.New(), storeID, adminName, adminEmail, string(hash), adminRole, adminStatus)
	return err
}
