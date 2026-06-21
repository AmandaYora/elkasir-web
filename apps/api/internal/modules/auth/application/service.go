// Package application holds the auth module's use cases (login, refresh, logout, me).
package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/auth/domain"
	"github.com/elkasir/api/internal/modules/auth/infrastructure"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
	"github.com/elkasir/api/internal/platform/security"
)

type Service struct {
	q   *sqlcgen.Queries
	mgr *infrastructure.Manager
}

func NewService(q *sqlcgen.Queries, mgr *infrastructure.Manager) *Service {
	return &Service{q: q, mgr: mgr}
}

func invalidCreds() error    { return httpx.Unauthorized("Email/username atau password salah.") }
func inactiveAccount() error { return httpx.Forbidden("Akun ini nonaktif. Hubungi pemilik.") }
func invalidSession() error  { return httpx.Unauthorized("Sesi tidak valid. Silakan login ulang.") }

// LoginAdmin validates admin credentials and issues a token pair.
func (s *Service) LoginAdmin(ctx context.Context, email, password string) (domain.TokenPair, domain.Identity, error) {
	u, err := s.q.GetAdminUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return domain.TokenPair{}, domain.Identity{}, invalidCreds()
	}
	if u.Status != sqlcgen.AdminUsersStatusActive {
		return domain.TokenPair{}, domain.Identity{}, inactiveAccount()
	}
	if !security.VerifyPassword(u.PasswordHash, password) {
		return domain.TokenPair{}, domain.Identity{}, invalidCreds()
	}

	p := authcontract.Principal{SubjectID: u.ID, StoreID: u.StoreID, Actor: authcontract.ActorAdmin, Role: string(u.Role)}
	pair, err := s.issuePair(ctx, p)
	if err != nil {
		return domain.TokenPair{}, domain.Identity{}, err
	}
	_ = s.q.TouchAdminUserLastActive(ctx, sqlcgen.TouchAdminUserLastActiveParams{
		LastActiveAt: sql.NullTime{Time: time.Now().UTC(), Valid: true},
		ID:           u.ID,
	})
	return pair, domain.Identity{ID: u.ID, Name: u.Name, Email: u.Email, Role: string(u.Role), StoreID: u.StoreID, Actor: authcontract.ActorAdmin}, nil
}

// LoginStaff validates POS staff credentials and issues a token pair.
func (s *Service) LoginStaff(ctx context.Context, username, password string) (domain.TokenPair, domain.Identity, error) {
	st, err := s.q.GetStaffByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return domain.TokenPair{}, domain.Identity{}, invalidCreds()
	}
	if st.Status != sqlcgen.StaffStatusActive {
		return domain.TokenPair{}, domain.Identity{}, inactiveAccount()
	}
	if !security.VerifyPassword(st.PasswordHash, password) {
		return domain.TokenPair{}, domain.Identity{}, invalidCreds()
	}

	p := authcontract.Principal{SubjectID: st.ID, StoreID: st.StoreID, Actor: authcontract.ActorStaff, Role: string(st.Role)}
	pair, err := s.issuePair(ctx, p)
	if err != nil {
		return domain.TokenPair{}, domain.Identity{}, err
	}
	return pair, domain.Identity{ID: st.ID, Name: st.Name, Email: st.Email.String, Role: string(st.Role), StoreID: st.StoreID, Actor: authcontract.ActorStaff}, nil
}

// Refresh rotates the refresh token: revoke the old, issue a new pair.
func (s *Service) Refresh(ctx context.Context, rawRefresh string) (domain.TokenPair, error) {
	hash := hashToken(rawRefresh)
	rt, err := s.q.GetRefreshToken(ctx, hash)
	if err != nil {
		return domain.TokenPair{}, invalidSession()
	}
	if rt.RevokedAt.Valid || rt.ExpiresAt.Before(time.Now()) {
		return domain.TokenPair{}, invalidSession()
	}

	p, err := s.principalFromRefresh(ctx, rt)
	if err != nil {
		return domain.TokenPair{}, err
	}
	_ = s.q.RevokeRefreshToken(ctx, sqlcgen.RevokeRefreshTokenParams{
		RevokedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true},
		TokenHash: hash,
	})
	return s.issuePair(ctx, p)
}

// Logout revokes a refresh token (idempotent).
func (s *Service) Logout(ctx context.Context, rawRefresh string) error {
	return s.q.RevokeRefreshToken(ctx, sqlcgen.RevokeRefreshTokenParams{
		RevokedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true},
		TokenHash: hashToken(rawRefresh),
	})
}

// Me loads the current profile from the token principal.
func (s *Service) Me(ctx context.Context, p authcontract.Principal) (domain.Identity, error) {
	switch p.Actor {
	case authcontract.ActorAdmin:
		u, err := s.q.GetAdminUserByID(ctx, p.SubjectID)
		if err != nil {
			return domain.Identity{}, invalidSession()
		}
		return domain.Identity{ID: u.ID, Name: u.Name, Email: u.Email, Role: string(u.Role), StoreID: u.StoreID, Actor: authcontract.ActorAdmin}, nil
	case authcontract.ActorStaff:
		st, err := s.q.GetStaffByID(ctx, p.SubjectID)
		if err != nil {
			return domain.Identity{}, invalidSession()
		}
		return domain.Identity{ID: st.ID, Name: st.Name, Email: st.Email.String, Role: string(st.Role), StoreID: st.StoreID, Actor: authcontract.ActorStaff}, nil
	default:
		return domain.Identity{}, invalidSession()
	}
}

func (s *Service) principalFromRefresh(ctx context.Context, rt sqlcgen.RefreshToken) (authcontract.Principal, error) {
	switch rt.Actor {
	case sqlcgen.RefreshTokensActorAdmin:
		u, err := s.q.GetAdminUserByID(ctx, rt.SubjectID)
		if err != nil || u.Status != sqlcgen.AdminUsersStatusActive {
			return authcontract.Principal{}, invalidSession()
		}
		return authcontract.Principal{SubjectID: u.ID, StoreID: u.StoreID, Actor: authcontract.ActorAdmin, Role: string(u.Role)}, nil
	case sqlcgen.RefreshTokensActorStaff:
		st, err := s.q.GetStaffByID(ctx, rt.SubjectID)
		if err != nil || st.Status != sqlcgen.StaffStatusActive {
			return authcontract.Principal{}, invalidSession()
		}
		return authcontract.Principal{SubjectID: st.ID, StoreID: st.StoreID, Actor: authcontract.ActorStaff, Role: string(st.Role)}, nil
	default:
		return authcontract.Principal{}, invalidSession()
	}
}

func (s *Service) issuePair(ctx context.Context, p authcontract.Principal) (domain.TokenPair, error) {
	access, _, err := s.mgr.IssueAccess(p)
	if err != nil {
		return domain.TokenPair{}, err
	}
	raw, hash := newRefreshToken()
	actor := sqlcgen.RefreshTokensActorAdmin
	if p.Actor == authcontract.ActorStaff {
		actor = sqlcgen.RefreshTokensActorStaff
	}
	if err := s.q.CreateRefreshToken(ctx, sqlcgen.CreateRefreshTokenParams{
		ID:        id.New(),
		Actor:     actor,
		SubjectID: p.SubjectID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(s.mgr.RefreshTTL()),
	}); err != nil {
		return domain.TokenPair{}, err
	}
	return domain.TokenPair{AccessToken: access, RefreshToken: raw, ExpiresIn: int64(s.mgr.AccessTTL().Seconds())}, nil
}

func newRefreshToken() (raw, hash string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	raw = hex.EncodeToString(b)
	return raw, hashToken(raw)
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
