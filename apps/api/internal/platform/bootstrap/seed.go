// Package bootstrap mengisi data awal MINIMAL: 1 toko + setting default + 1 akun
// admin (email "admin", password "admin123"). Idempoten — aman dijalankan berulang.
// Dipakai bersama oleh cmd/seed dan subcommand `api seed`.
package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"

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

	// PlatformEmail / PlatformPassword adalah kredensial superadmin bootstrap — WAJIB diganti
	// setelah login pertama di lingkungan produksi.
	platformName     = "superadmin"
	PlatformEmail    = "superadmin@gmail.com"
	PlatformPassword = "superadmin"

	maxDiscountPercent    = 10
	maxOperationalExpense = 200000
	cashVarianceTolerance = 5000
)

// planSeed adalah katalog paket langganan platform (bukan data tenant — sama untuk semua
// store), diisi idempoten lewat ON DUPLICATE KEY UPDATE seperti settings/admin di atas.
type planSeed struct {
	code       string
	name       string
	price      int64
	periodDays int
}

var subscriptionPlanSeeds = []planSeed{
	{code: "bulanan", name: "Paket 1 Bulan", price: 299000, periodDays: 30},
	{code: "tahunan", name: "Paket 1 Tahun", price: 2490000, periodDays: 365},
}

// Seed menjalankan seluruh upsert data awal dalam satu transaksi. Bootstrap toko contoh
// (store + settings + admin default) HANYA dijalankan bila database masih benar-benar kosong
// (belum ada store sama sekali) — sebelumnya ensureStore mencari store By name = "Elkasir",
// yang berarti menjalankan seed ini di database produksi yang sudah punya tenant asli (nama
// apa pun selain "Elkasir") akan diam-diam membuat toko "Elkasir" KEDUA plus admin cadangan
// "admin"/"admin123" ber-role owner menempel ke toko itu — melanggar asumsi "satu tenant" dan
// menanam kredensial default yang dikenal publik. Katalog paket (referensi, aman lintas store)
// dan akun superadmin platform (idempoten by email) tetap SELALU di-upsert di kedua kondisi.
func Seed(ctx context.Context, pool *sql.DB) error {
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	hasStore, err := anyStoreExists(ctx, tx)
	if err != nil {
		return err
	}
	if !hasStore {
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
	}
	if err := upsertSubscriptionPlans(ctx, tx); err != nil {
		return err
	}
	if err := upsertPlatformUser(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func anyStoreExists(ctx context.Context, tx *sql.Tx) (bool, error) {
	var one int
	err := tx.QueryRowContext(ctx, "SELECT 1 FROM stores LIMIT 1").Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func ensureStore(ctx context.Context, tx *sql.Tx) (string, error) {
	storeID := id.New()
	_, err := tx.ExecContext(ctx,
		"INSERT INTO stores (id, name, slug, type, timezone) VALUES (?, ?, ?, ?, ?)",
		storeID, storeName, slugify(storeName), storeType, storeTZ)
	return storeID, err
}

// upsertPlatformUser mengisi satu akun superadmin bootstrap (data platform, bukan data tenant).
func upsertPlatformUser(ctx context.Context, tx *sql.Tx) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(PlatformPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO platform_users (id, name, email, password_hash)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name = VALUES(name), password_hash = VALUES(password_hash)`,
		id.New(), platformName, PlatformEmail, string(hash))
	return err
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

// upsertSubscriptionPlans mengisi katalog paket langganan (subscription module) — data
// referensi platform, bukan data tenant, sehingga aman di-upsert per `code` lintas store.
func upsertSubscriptionPlans(ctx context.Context, tx *sql.Tx) error {
	for _, p := range subscriptionPlanSeeds {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO subscription_plans (id, code, name, price, period_days)
			VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				name = VALUES(name), price = VALUES(price), period_days = VALUES(period_days)`,
			id.New(), p.code, p.name, p.price, p.periodDays)
		if err != nil {
			return err
		}
	}
	return nil
}

var slugInvalidChars = regexp.MustCompile(`[^a-z0-9]+`)

// slugify menurunkan slug default dari nama toko (bootstrap store). Tenant baru lewat
// `platform` module memilih slug-nya sendiri secara eksplisit — ini hanya dipakai di sini.
func slugify(name string) string {
	return strings.Trim(slugInvalidChars.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// ProvisionTenantInput menjelaskan tenant baru + akun owner pertamanya.
type ProvisionTenantInput struct {
	StoreName     string
	StoreSlug     string
	OwnerName     string
	OwnerEmail    string
	OwnerPassword string
}

// ProvisionTenant membuat tenant BARU (store + settings default + akun admin owner) dalam
// satu transaksi — primitif onboarding tenant milik modul `platform`. Berbeda dari Seed (data
// bootstrap idempoten), ini SELALU membuat baris baru; slug/email yang bentrok muncul sebagai
// error unique-constraint biasa (db.IsDuplicate) untuk dipetakan pemanggil ke 409.
func ProvisionTenant(ctx context.Context, pool *sql.DB, in ProvisionTenantInput) (string, error) {
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	storeID := id.New()
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO stores (id, name, slug, type, timezone) VALUES (?, ?, ?, ?, ?)",
		storeID, in.StoreName, in.StoreSlug, storeType, storeTZ); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO settings (id, store_id, max_discount_percent, max_operational_expense, cash_variance_tolerance)
		VALUES (?, ?, ?, ?, ?)`,
		id.New(), storeID, maxDiscountPercent, maxOperationalExpense, cashVarianceTolerance); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.OwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO admin_users (id, store_id, name, email, password_hash, role, status)
		VALUES (?, ?, ?, ?, ?, 'owner', 'active')`,
		id.New(), storeID, in.OwnerName, in.OwnerEmail, string(hash)); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return storeID, nil
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
