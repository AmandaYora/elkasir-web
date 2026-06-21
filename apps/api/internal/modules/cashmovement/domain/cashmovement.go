// Package domain holds the cashmovement module's entities, value objects, and rules.
package domain

import (
	"strings"
	"time"

	"github.com/elkasir/api/internal/platform/httpx"
)

// Cash movement types (modal, biaya, penyesuaian).
const (
	TypeCapital    = "capital"
	TypeExpense    = "expense"
	TypeAdjustment = "adjustment"
)

// CashMovement is the cash movement read model.
type CashMovement struct {
	ID         string
	ShiftID    string
	Type       string
	Amount     int64
	Notes      string
	CreatedBy  string
	ApprovedBy string
	CreatedAt  time.Time
}

// Input adalah payload pembuatan kas masuk/keluar (sudah didekode dari JSON).
type Input struct {
	Type       string `json:"type"`
	Amount     int64  `json:"amount"`
	Notes      string `json:"notes"`
	ApprovedBy string `json:"approvedBy"`
}

// Validate enforces cash movement business rules (type + amount).
func (in Input) Validate() error {
	if !ValidType(in.Type) {
		return httpx.Validation("Tipe harus 'capital', 'expense', atau 'adjustment'.")
	}
	// Modal & biaya wajib > 0; penyesuaian boleh negatif (asal tidak nol).
	if in.Type == TypeAdjustment {
		if in.Amount == 0 {
			return httpx.Validation("Nominal penyesuaian tidak boleh nol.")
		}
	} else if in.Amount <= 0 {
		return httpx.Validation("Nominal harus lebih dari 0.")
	}
	return nil
}

// ValidType reports whether s is a recognized cash movement type.
func ValidType(s string) bool {
	switch s {
	case TypeCapital, TypeExpense, TypeAdjustment:
		return true
	default:
		return false
	}
}

// ListFilter holds the cash movement listing filters.
type ListFilter struct {
	StoreID string
	Limit   int
	Offset  int
}

// TrimmedApprovedBy returns the approver with surrounding whitespace removed.
func (in Input) TrimmedApprovedBy() string { return strings.TrimSpace(in.ApprovedBy) }
