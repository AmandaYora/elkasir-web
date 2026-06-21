// Package security holds domain-agnostic crypto utilities (password hashing).
package security

import "golang.org/x/crypto/bcrypt"

// HashPassword produces a bcrypt hash.
func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(h), err
}

// VerifyPassword compares a password to a bcrypt hash (constant-time).
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
