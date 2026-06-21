// Package domain holds the transaction module's pure value objects and listing filters.
//
// The transaction read-model itself is the sqlc-generated row (sqlcgen.Transaction); the
// line-item snapshot value type used across modules lives in the module contract
// (salesclient.SaleItem). The create/read DTOs (with JSON tags) live in the application
// layer. This file holds the pure value/filter types that are free of transport concerns.
package domain

import "time"

// ListFilter untuk daftar transaksi (filter opsional).
type ListFilter struct {
	StoreID       string
	Status        string
	Source        string
	PaymentMethod string
	Search        string
	From          *time.Time
	To            *time.Time
	Limit         int
	Offset        int
}
