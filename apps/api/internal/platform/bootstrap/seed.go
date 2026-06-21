// Package bootstrap mengisi data awal MINIMAL: 1 toko + setting default + 1 akun
// admin (email "admin", password "admin123"). Idempoten — aman dijalankan berulang.
// Dipakai bersama oleh cmd/seed dan subcommand `api seed`.
package bootstrap

import (
	"context"
	"database/sql"
	"errors"

	"github.com/elkasir/api/internal/platform/id"
	"golang.org/x/crypto/bcrypt"
)

const (
	storeName = "Elkasir"
	storeType = "Restoran"
	storeTZ   = "Asia/Jakarta"

	adminName  = "Administrator"
	adminRole  = "owner"
	adminStatus = "active"

	// AdminEmail / AdminUsername / AdminPassword adalah kredensial bootstrap awal —
	// WAJIB diganti setelah login pertama di lingkungan produksi.
	AdminEmail    = "admin"
	AdminUsername = "admin"
	AdminPassword = "admin123"

	maxDiscountPercent    = 10
	maxOperationalExpense = 200000
	cashVarianceTolerance = 5000
)

// Seed menjalankan seluruh upsert data awal dalam satu transaksi.
func Seed(ctx context.Context, pool *sql.DB) error {
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
	return tx.Commit()
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
	hash, err := bcrypt.GenerateFromPassword([]byte(AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO admin_users (id, store_id, name, email, username, password_hash, role, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			store_id = VALUES(store_id), name = VALUES(name), username = VALUES(username),
			password_hash = VALUES(password_hash), role = VALUES(role), status = VALUES(status)`,
		id.New(), storeID, adminName, AdminEmail, AdminUsername, string(hash), adminRole, adminStatus)
	return err
}
