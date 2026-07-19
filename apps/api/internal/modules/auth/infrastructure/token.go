// Package infrastructure holds the auth module's technical adapters: JWT manager,
// password hashing, and the HTTP authentication middleware.
package infrastructure

import (
	"errors"
	"time"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned when an access token fails validation.
var ErrInvalidToken = errors.New("token tidak valid")

type claims struct {
	StoreID string `json:"store_id"`
	Actor   string `json:"actor"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

// Manager issues & validates access tokens (HS256).
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (m *Manager) AccessTTL() time.Duration  { return m.accessTTL }
func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

// IssueAccess issues an access token with the standard human-session TTL.
func (m *Manager) IssueAccess(p authcontract.Principal) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.accessTTL)
	c := claims{
		StoreID: p.StoreID,
		Actor:   string(p.Actor),
		Role:    p.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   p.SubjectID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
	return signed, exp, err
}

// ParseAccess validates a token and returns the Principal.
func (m *Manager) ParseAccess(token string) (authcontract.Principal, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil {
		return authcontract.Principal{}, ErrInvalidToken
	}
	return authcontract.Principal{
		SubjectID: c.Subject,
		StoreID:   c.StoreID,
		Actor:     authcontract.Actor(c.Actor),
		Role:      c.Role,
	}, nil
}
