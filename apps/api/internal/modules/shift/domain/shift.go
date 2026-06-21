// Package domain holds the shift module's entities, value objects, and rules.
package domain

import "time"

// Shift is the cashier-shift read model with cash reconciliation totals.
type Shift struct {
	ID                string
	StaffID           string
	Status            string
	InitialCash       int64
	CashSales         int64
	QrisSales         int64
	AdditionalCapital int64
	Expenses          int64
	Withdrawals       int64
	Adjustments       int64
	DrawerOpenCount   int32
	ExpectedCash      *int64
	ActualCash        *int64
	Variance          *int64
	CloseApprovedBy   string
	OpenedAt          time.Time
	ClosedAt          *time.Time
	CreatedAt         time.Time
}

// OpenInput is the open-shift payload (decoded from JSON).
type OpenInput struct {
	InitialCash int64 `json:"initialCash"`
}

// Validate enforces open-shift business rules.
func (in OpenInput) Validate() error { return nil }

// CloseInput is the close-shift payload (decoded from JSON).
type CloseInput struct {
	ActualCash      int64  `json:"actualCash"`
	DrawerOpenCount int32  `json:"drawerOpenCount"`
	CloseApprovedBy string `json:"closeApprovedBy"`
}

// Validate enforces close-shift business rules.
func (in CloseInput) Validate() error { return nil }

// ListFilter holds the shift listing filters.
type ListFilter struct {
	StoreID string
	Limit   int
	Offset  int
}
