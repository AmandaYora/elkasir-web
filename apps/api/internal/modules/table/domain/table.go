// Package domain holds the table module's entities, value objects, and rules.
package domain

import (
	"strings"
	"time"

	"github.com/elkasir/api/internal/platform/httpx"
)

// Table is the dining-table read model (master data entry).
type Table struct {
	ID        string
	Code      string
	Name      string
	Area      string
	Seats     int32
	Status    string
	CreatedAt time.Time
}

// Input is the create/update payload (decoded from JSON).
type Input struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Area   string `json:"area"`
	Seats  int32  `json:"seats"`
	Status string `json:"status"`
}

// Validate enforces table business rules.
func (in Input) Validate() error {
	if strings.TrimSpace(in.Code) == "" {
		return httpx.Validation("Kode meja wajib diisi.")
	}
	if in.Seats < 0 {
		return httpx.Validation("Jumlah kursi tidak boleh negatif.")
	}
	if in.Status != "" && in.Status != "active" && in.Status != "inactive" {
		return httpx.Validation("Status harus 'active' atau 'inactive'.")
	}
	return nil
}

// ListFilter holds the table listing filters.
type ListFilter struct {
	StoreID string
}
