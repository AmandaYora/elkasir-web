package uow

import (
	"context"
	"errors"
	"testing"

	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Q tanpa transaksi aktif harus mengembalikan Queries dasar (pool).
func TestQ_NoTxReturnsBase(t *testing.T) {
	m := &Manager{base: sqlcgen.New(nil)}
	if got := m.Q(context.Background()); got != m.base {
		t.Fatalf("Q tanpa tx harus mengembalikan base Queries")
	}
}

// Q dengan transaksi aktif di ctx harus mengembalikan Queries ber-transaksi tsb.
func TestQ_WithTxReturnsTxQueries(t *testing.T) {
	m := &Manager{base: sqlcgen.New(nil)}
	txq := sqlcgen.New(nil) // sentinel berbeda dari base
	ctx := context.WithValue(context.Background(), ctxKey{}, txq)
	if got := m.Q(ctx); got != txq {
		t.Fatalf("Q harus mengembalikan Queries dari ctx, bukan base")
	}
}

// Run saat sudah berada dalam transaksi harus "gabung": memanggil fn langsung tanpa
// menyentuh db (db nil membuktikan BeginTx tak dipanggil), dan meneruskan error fn.
func TestRun_JoinsExistingTx(t *testing.T) {
	m := &Manager{db: nil, base: sqlcgen.New(nil)} // db nil: BeginTx akan panic bila dipanggil
	ctx := context.WithValue(context.Background(), ctxKey{}, sqlcgen.New(nil))

	called := false
	sentinel := errors.New("dari fn")
	err := m.Run(ctx, func(ctx context.Context) error {
		called = true
		// Q di dalam fn harus tetap melihat transaksi yang sama (gabung).
		if _, ok := ctx.Value(ctxKey{}).(*sqlcgen.Queries); !ok {
			t.Fatalf("ctx di dalam fn harus tetap membawa transaksi")
		}
		return sentinel
	})
	if !called {
		t.Fatalf("fn harus dipanggil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("Run harus meneruskan error fn, dapat: %v", err)
	}
}
