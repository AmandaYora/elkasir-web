// Package domain holds the auth module's domain value objects.
package domain

import authcontract "github.com/elkasir/api/internal/modules/auth/contracts"

// Identity is the compact profile returned after login / on /me.
type Identity struct {
	ID      string
	Name    string
	Email   string
	Role    string
	StoreID string
	Actor   authcontract.Actor
}

// TokenPair is the result of issuing tokens.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds (access-token lifetime)
}
