// Package tableclient adalah KONTRAK modul table (meja) untuk dikonsumsi modul selforder
// saat memvalidasi meja dari kode QR dan menampilkan info meja.
package tableclient

import (
	"context"
	"errors"
)

// Table adalah ringkasan meja lintas-modul (status sebagai string netral).
type Table struct {
	ID      string
	StoreID string
	Code    string
	Name    string
	Area    string
	Status  string // "active" | "inactive" (DiningTablesStatus)
}

// Client adalah kontrak yang dipublikasikan modul table.
type Client interface {
	FindByCode(ctx context.Context, code string) (Table, error)     // entry self-order; store ditemukan dari code
	GetByID(ctx context.Context, storeID, id string) (Table, error) // detail meja
	ListAll(ctx context.Context, storeID string) ([]Table, error)   // peta meja (mis. daftar pesanan masuk)
}

var ErrNotFound = errors.New("meja tidak ditemukan")
