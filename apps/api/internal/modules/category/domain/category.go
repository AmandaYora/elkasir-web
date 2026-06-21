// Package domain holds the category module's entities, value objects, and rules.
package domain

import (
	"strings"
	"time"

	"github.com/elkasir/api/internal/platform/httpx"
)

// Category is the category read model (master data + resolved product count).
type Category struct {
	ID           string
	Name         string
	SortOrder    int32
	ProductCount int64
	CreatedAt    time.Time
}

// Input is the create/update payload (decoded from JSON).
type Input struct {
	Name      string `json:"name"`
	SortOrder int32  `json:"sortOrder"`
}

// Validate enforces category business rules.
func (in Input) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return httpx.Validation("Nama kategori wajib diisi.")
	}
	return nil
}

// ListFilter holds the category listing filters.
type ListFilter struct {
	StoreID string
}
