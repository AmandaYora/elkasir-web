// Package domain holds the platformuser module's entities and rules — superadmin
// (platform_users) account management, consumed by `platform` via platformuserclient.Client.
package domain

import (
	"strings"
	"time"

	"github.com/elkasir/api/internal/platform/httpx"
)

// PlatformUser is the platformuser read model.
type PlatformUser struct {
	ID        string
	Name      string
	Email     string
	Status    string
	CreatedAt time.Time
}

// CreateInput is the create-superadmin payload.
type CreateInput struct {
	Name     string
	Email    string
	Password string
}

// Validate enforces platform-user creation rules.
func (in CreateInput) Validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return httpx.Validation("Nama wajib diisi.")
	}
	if strings.TrimSpace(in.Email) == "" {
		return httpx.Validation("Email wajib diisi.")
	}
	if len(in.Password) < 6 {
		return httpx.Validation("Password minimal 6 karakter.")
	}
	return nil
}
