// Package adminuserclient is the PUBLIC contract of the adminuser module — consumed by the
// `platform` module for the superadmin cross-tenant admin-password-reset flow (the ONE
// deliberate cross-tenant exception; see platform.module.go). adminuser keeps its own
// /admin-users/* routes (self-service, store-scoped) for everything else.
package adminuserclient

import "context"

// AdminUser is the minimal admin-account read model exposed cross-module (no password hash).
type AdminUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Client is the contract published by the adminuser module.
type Client interface {
	// ListByStore returns a tenant's admin accounts (so a caller can pick which one to reset).
	ListByStore(ctx context.Context, storeID string) ([]AdminUser, error)
	// ResetPassword resets one admin account's password within a store.
	ResetPassword(ctx context.Context, storeID, uid, newPassword string) error
}
