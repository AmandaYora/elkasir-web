// Package domain holds the report module's value types and filters (read-only analytics).
// The report module has NO contract and performs NO writes; these are plain structs used
// to shape analytics queries. JSON DTOs live in the application layer.
package domain

import "time"

// DateRange is the inclusive analytics window applied to every report query.
type DateRange struct {
	From time.Time
	To   time.Time
}
