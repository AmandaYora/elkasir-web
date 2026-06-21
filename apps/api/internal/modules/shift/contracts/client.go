// Package shiftclient adalah KONTRAK modul shift untuk dikonsumsi modul lain
// (transaction/selforder) saat menautkan penjualan ke shift terbuka.
package shiftclient

import "context"

// Client adalah kontrak yang dipublikasikan modul shift.
type Client interface {
	// CurrentOpenID mengembalikan ID shift yang sedang terbuka untuk store,
	// atau string kosong bila tidak ada shift terbuka.
	CurrentOpenID(ctx context.Context, storeID string) (string, error)
}
