// Package uow menyediakan Unit of Work: satu transaksi DB yang "ambient" di context.
//
// Pola modular-monolith (lihat docs/architecture/modular-monolith.md §5):
//   - Setiap repo/module-client memanggil Q(ctx) untuk SELURUH query sqlc, sehingga
//     otomatis ikut transaksi aktif bila ada (jika tidak, jalan di pool/auto-commit).
//   - Orchestrator membungkus tulis lintas-modul dengan Run(...) agar atomik
//     (mis. kurangi stok + buat transaksi + tandai order dalam satu transaksi DB).
//
// Tipe transaksi TIDAK bocor ke kontrak antar-modul: ia hanya hidup di dalam context.
package uow

import (
	"context"
	"database/sql"

	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// ctxKey adalah kunci privat untuk menyimpan Queries ber-transaksi di context.
type ctxKey struct{}

// Manager memegang pool DB dan Queries dasar (berbasis pool).
type Manager struct {
	db   *sql.DB
	base *sqlcgen.Queries
}

// New membuat Manager dari pool DB.
func New(db *sql.DB) *Manager {
	return &Manager{db: db, base: sqlcgen.New(db)}
}

// Q mengembalikan Queries ber-transaksi bila ada transaksi aktif di ctx; jika tidak,
// memakai Queries berbasis pool (auto-commit per statement).
func (m *Manager) Q(ctx context.Context) *sqlcgen.Queries {
	if q, ok := ctx.Value(ctxKey{}).(*sqlcgen.Queries); ok && q != nil {
		return q
	}
	return m.base
}

// DB mengembalikan pool, untuk query tulis-tangan read-only di luar transaksi.
func (m *Manager) DB() *sql.DB { return m.db }

// Run menjalankan fn dalam SATU transaksi DB. Transaksi dibawa di ctx sehingga semua
// pemanggilan Q(ctx) di dalam fn ikut transaksi yang sama. Bila ctx sudah berada dalam
// transaksi (nested), fn ikut transaksi yang ada tanpa membuka transaksi baru.
// Commit bila fn sukses; rollback bila fn mengembalikan error atau panic.
func (m *Manager) Run(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := ctx.Value(ctxKey{}).(*sqlcgen.Queries); ok {
		return fn(ctx) // sudah dalam transaksi → gabung (tidak membuka transaksi baru)
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	txCtx := context.WithValue(ctx, ctxKey{}, m.base.WithTx(tx))
	if err = fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
