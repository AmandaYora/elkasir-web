// Package productclient is the PUBLIC contract of the product module for other modules
// (e.g. transaction/selforder) — interface + DTO + sentinel errors. It does NOT import
// the product implementation, so it is free of import cycles.
package productclient

import (
	"context"
	"errors"
)

// ProductSale is a product summary for selling. Price & category are snapshotted when a
// transaction is created (immune to later product edits).
type ProductSale struct {
	ID       string
	Name     string
	Category string
	Price    int64
	ImageURL string // only set by ListActive (menu)
	Active   bool
}

// Client is the contract published by the product module.
type Client interface {
	GetForSale(ctx context.Context, storeID, productID string) (ProductSale, error) // getById()
	ListActive(ctx context.Context, storeID string) ([]ProductSale, error)          // search() / menu
	Decrease(ctx context.Context, storeID, productID string, qty int32) error       // decrease() — tx-aware
	// Increase atomically restocks (e.g. when a sale is voided) — tx-aware. Best-effort: a
	// missing product (deleted since the sale) is a no-op, not an error.
	Increase(ctx context.Context, storeID, productID string, qty int32) error
}

var (
	// ErrNotFound: product does not exist for the store.
	ErrNotFound = errors.New("produk tidak ditemukan")
	// ErrInactive: product exists but is inactive (not sellable).
	ErrInactive = errors.New("produk nonaktif")
	// ErrInsufficientStock: stock < requested qty on Decrease.
	ErrInsufficientStock = errors.New("stok tidak cukup")
)
