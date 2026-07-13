// Package platformuserclient is the PUBLIC contract of the platformuser module — consumed by
// the `platform` module (superadmin user-management surface, PLAN.md §2.9). This module has NO
// HTTP handler and NO routes of its own — `platform` owns the /platform/users/* routes and
// reaches this module only through this contract, the same "contracts-only" pattern already
// used by `payment` (see payment.module.go).
package platformuserclient

import (
	"context"
	"time"
)

// PlatformUser is the superadmin account read model (no password hash).
type PlatformUser struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateInput is the create-superadmin payload.
type CreateInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Client is the contract published by the platformuser module.
type Client interface {
	List(ctx context.Context) ([]PlatformUser, error)
	Create(ctx context.Context, in CreateInput) (PlatformUser, error)
	// SetStatus rejects if actingUserID == targetID && status == "inactive" (§2.9 — a
	// superadmin cannot deactivate their own account).
	SetStatus(ctx context.Context, actingUserID, targetID, status string) error
	ResetPassword(ctx context.Context, id, newPassword string) error
}
