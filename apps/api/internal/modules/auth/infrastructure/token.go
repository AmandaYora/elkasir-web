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
	secret      []byte
	accessTTL   time.Duration
	refreshTTL  time.Duration
	appTokenTTL time.Duration // ActorApp only (§10.1.3) — no refresh token for this actor
}

func NewManager(secret string, accessTTL, refreshTTL, appTokenTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL, appTokenTTL: appTokenTTL}
}

func (m *Manager) AccessTTL() time.Duration   { return m.accessTTL }
func (m *Manager) RefreshTTL() time.Duration  { return m.refreshTTL }
func (m *Manager) AppTokenTTL() time.Duration { return m.appTokenTTL }

// IssueAccess issues an access token with the standard human-session TTL.
func (m *Manager) IssueAccess(p authcontract.Principal) (string, time.Time, error) {
	return m.IssueAccessWithTTL(p, m.accessTTL)
}

// IssueAccessWithTTL issues an access token with an explicit TTL override — used for ActorApp
// (PLAN.md §10.1.3: a separate, shorter-lived, no-refresh-token machine-credential lifetime,
// distinct from the human accessTTL/refreshTTL pair this Manager was constructed with).
func (m *Manager) IssueAccessWithTTL(p authcontract.Principal, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(ttl)
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
