// Package migrator menjalankan migrasi skema (golang-migrate) memakai berkas yang
// disematkan via go:embed. Dengan ini binary tunggal bisa migrate sekaligus serve —
// tidak perlu image migrate terpisah pada runtime distroless.
package migrator

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/elkasir/api/db/migrations"
	"github.com/golang-migrate/migrate/v4"
	migmysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "github.com/go-sql-driver/mysql"
)

// newMigrator membangun instance migrate dari sumber embed + koneksi MySQL.
// DSN yang dipakai sama persis dengan aplikasi (sudah memuat multiStatements=true),
// sehingga berkas migrasi multi-statement berjalan benar.
func newMigrator(dsn string) (*migrate.Migrate, *sql.DB, error) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, nil, fmt.Errorf("migrator: source: %w", err)
	}
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("migrator: open: %w", err)
	}
	drv, err := migmysql.WithInstance(pool, &migmysql.Config{})
	if err != nil {
		_ = pool.Close()
		return nil, nil, fmt.Errorf("migrator: driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "mysql", drv)
	if err != nil {
		_ = pool.Close()
		return nil, nil, fmt.Errorf("migrator: instance: %w", err)
	}
	return m, pool, nil
}

// Up menerapkan seluruh migrasi yang belum dijalankan. ErrNoChange bukan error.
func Up(dsn string) error {
	m, pool, err := newMigrator(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = pool.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrator: up: %w", err)
	}
	return nil
}

// Down memundurkan n langkah migrasi (n > 0). ErrNoChange bukan error.
func Down(dsn string, n int) error {
	if n <= 0 {
		return fmt.Errorf("migrator: down butuh n > 0, dapat %d", n)
	}
	m, pool, err := newMigrator(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = pool.Close() }()
	if err := m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrator: down %d: %w", n, err)
	}
	return nil
}
