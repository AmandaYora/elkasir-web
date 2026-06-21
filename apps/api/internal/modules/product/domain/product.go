// Package domain holds the product module's entities, value objects, and rules.
package domain

import (
	"strings"
	"time"

	"github.com/elkasir/api/internal/platform/httpx"
)

// Product is the product read model (catalog entry + resolved category name).
type Product struct {
	ID         string
	CategoryID string
	Category   string
	Sku        string
	Name       string
	Price      int64
	Cost       int64
	Stock      int32
	Status     string
	ImageURL   string
	CreatedAt  time.Time
}

// Input is the create/update payload (decoded from JSON).
type Input struct {
	CategoryID string `json:"categoryId"`
	Sku        string `json:"sku"`
	Name       string `json:"name"`
	Price      int64  `json:"price"`
	Cost       int64  `json:"cost"`
	Stock      int32  `json:"stock"`
	Status     string `json:"status"`
	ImageURL   string `json:"imageUrl"`
}

// Validate enforces product business rules.
func (in Input) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return httpx.Validation("Nama produk wajib diisi.")
	}
	if in.Price < 0 || in.Cost < 0 || in.Stock < 0 {
		return httpx.Validation("Harga, modal, dan stok tidak boleh negatif.")
	}
	if in.Status != "" && in.Status != "active" && in.Status != "inactive" {
		return httpx.Validation("Status harus 'active' atau 'inactive'.")
	}
	return nil
}

// ListFilter holds the product listing filters.
type ListFilter struct {
	StoreID    string
	Status     string
	CategoryID string
	Search     string
	Limit      int
	Offset     int
}
