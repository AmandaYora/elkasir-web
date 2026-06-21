// Package domain holds the staff module's entities, value objects, and rules.
package domain

import (
	"strings"
	"time"

	"github.com/elkasir/api/internal/platform/httpx"
)

// Staff is the staff read model (POS account: cashier/supervisor).
type Staff struct {
	ID        string
	Name      string
	Username  string
	Email     string
	Role      string
	Status    string
	CreatedAt time.Time
}

// CreateInput is the staff creation payload (decoded from JSON).
type CreateInput struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

// UpdateInput is the staff update payload (without password).
type UpdateInput struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

// Validate enforces staff creation business rules (name, username, password).
func (in CreateInput) Validate() error {
	if err := validateNameUsername(in.Name, in.Username); err != nil {
		return err
	}
	if len(in.Password) < 6 {
		return httpx.Validation("Password minimal 6 karakter.")
	}
	return nil
}

// Validate enforces staff update business rules (name, username).
func (in UpdateInput) Validate() error {
	return validateNameUsername(in.Name, in.Username)
}

func validateNameUsername(name, username string) error {
	if strings.TrimSpace(name) == "" {
		return httpx.Validation("Nama wajib diisi.")
	}
	if strings.TrimSpace(username) == "" {
		return httpx.Validation("Username wajib diisi.")
	}
	return nil
}

// ListFilter holds the staff listing filters.
type ListFilter struct {
	StoreID string
}
