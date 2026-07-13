// Package domain holds the withdrawal module's entities, value objects, and rules.
package domain

import (
	"strings"
	"time"

	"github.com/elkasir/api/internal/platform/httpx"
)

// Withdrawal is the withdrawal read model (store balance disbursement request).
type Withdrawal struct {
	ID             string
	StoreID        string
	Amount         int64
	Bank           string
	Account        string
	Holder         string
	Status         string
	Reference      string
	RequestedBy    string
	ProcessedBy    string
	ClaimedAt      *time.Time
	ProcessedAt    *time.Time
	RejectedReason string
	CreatedAt      time.Time
}

// Input is the withdrawal request payload (decoded from JSON).
type Input struct {
	Amount  int64  `json:"amount"`
	Bank    string `json:"bank"`
	Account string `json:"account"`
	Holder  string `json:"holder"`
}

// Validate enforces withdrawal business rules.
func (in Input) Validate() error {
	if in.Amount <= 0 {
		return httpx.Validation("Jumlah pencairan harus lebih dari nol.")
	}
	if strings.TrimSpace(in.Bank) == "" || strings.TrimSpace(in.Account) == "" || strings.TrimSpace(in.Holder) == "" {
		return httpx.Validation("Bank, nomor rekening, dan nama pemilik wajib diisi.")
	}
	return nil
}

// ListFilter holds the withdrawal listing filters.
type ListFilter struct {
	StoreID string
	Limit   int32
	Offset  int32
}
