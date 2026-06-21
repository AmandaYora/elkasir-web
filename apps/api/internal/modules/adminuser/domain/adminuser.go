// Package domain holds the adminuser module's entities, value objects, and rules.
package domain

import (
	"strings"
	"time"

	"github.com/elkasir/api/internal/platform/httpx"
)

// AdminUser is the admin dashboard user read model (owner/admin/manager/viewer).
type AdminUser struct {
	ID           string
	Name         string
	Email        string
	Username     string
	Role         string
	Status       string
	LastActiveAt *time.Time
	CreatedAt    time.Time
}

// CreateInput is the admin user creation payload (decoded from JSON).
type CreateInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

// UpdateInput is the admin user update payload (decoded from JSON).
type UpdateInput struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

// Validate enforces admin user creation business rules (name, email, username, password).
func (in CreateInput) Validate() error {
	if _, err := ValidateNameEmail(in.Name, in.Email); err != nil {
		return err
	}
	if _, err := ValidateUsername(in.Username); err != nil {
		return err
	}
	if len(in.Password) < 6 {
		return httpx.Validation("Password minimal 6 karakter.")
	}
	return nil
}

// ValidateUsername validates the username and returns it normalized (trimmed, lowercased).
// Allowed: 3–100 chars, huruf/angka dan . _ - (tanpa spasi) — selaras dengan login by username.
func ValidateUsername(username string) (string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" {
		return "", httpx.Validation("Username wajib diisi.")
	}
	if len(username) < 3 || len(username) > 100 {
		return "", httpx.Validation("Username harus 3–100 karakter.")
	}
	for _, r := range username {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '.' && r != '_' && r != '-' {
			return "", httpx.Validation("Username hanya boleh huruf, angka, titik, garis bawah, atau strip.")
		}
	}
	return username, nil
}

// Validate enforces admin user update business rules (name, email).
func (in UpdateInput) Validate() error {
	_, err := ValidateNameEmail(in.Name, in.Email)
	return err
}

// ValidateNameEmail validates name + email and returns the normalized (lowercased) email.
func ValidateNameEmail(name, email string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", httpx.Validation("Nama wajib diisi.")
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", httpx.Validation("Email wajib diisi.")
	}
	return email, nil
}

// ListFilter holds the admin user listing filters.
type ListFilter struct {
	StoreID string
}
