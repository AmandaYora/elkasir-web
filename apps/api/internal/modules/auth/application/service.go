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
	subscriptionclient "github.com/elkasir/api/internal/modules/subscription/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
	"github.com/elkasir/api/internal/platform/security"
)

type Service struct {
	q   *sqlcgen.Queries
	mgr *infrastructure.Manager
	sub subscriptionclient.Client // set post-construction — see auth.module.go SetSubscriptionClient
}

func NewService(q *sqlcgen.Queries, mgr *infrastructure.Manager) *Service {
	return &Service{q: q, mgr: mgr}
}

// SetSubscriptionClient wires the subscription contract for the package-inactive login gate
// (§2.15) — same post-construction pattern (and same reason) as Middleware.SetSubscriptionClient.
func (s *Service) SetSubscriptionClient(c subscriptionclient.Client) { s.sub = c }

// packageInactive reports "no active package" per §2.15 — mirrors Middleware.hasActivePackage.
// Skipped (never blocks) if the subscription client hasn't been wired yet (construction window).
func (s *Service) packageInactive(ctx context.Context, storeID string) (bool, error) {
	if s.sub == nil {
		return false, nil
	}
	sub, err := s.sub.Current(ctx, storeID)
	if err != nil {
		return false, err
	}
	if sub.Status != "active" || sub.CurrentPeriodEnd == nil {
		return true, nil
	}
	return sub.CurrentPeriodEnd.Before(time.Now()), nil
}

func packageInactiveErr() error {
	return httpx.Forbidden("Toko ini belum memiliki paket langganan aktif. Hubungi pemilik toko untuk memperbarui langganan.")
}

func invalidCreds() error    { return httpx.Unauthorized("Email/username atau password salah.") }
func inactiveAccount() error { return httpx.Forbidden("Akun ini nonaktif. Hubungi pemilik.") }
func invalidSession() error  { return httpx.Unauthorized("Sesi tidak valid. Silakan login ulang.") }
func suspendedTenant() error {
	return httpx.Forbidden("Toko Anda sedang dinonaktifkan. Hubungi platform.")
}

// tenantSuspended reads stores.status directly (§2.13/§2.14 shared-kernel exception) — used at
// login/refresh time, in addition to (not instead of) the per-request middleware check.
func (s *Service) tenantSuspended(ctx context.Context, storeID string) (bool, error) {
	status, err := s.q.GetStoreStatus(ctx, storeID)
	if err != nil {
		return false, err
	}
	return status == sqlcgen.StoresStatusSuspended, nil
}

// LoginAdmin validates admin credentials and issues a token pair.
// The identifier may be an email OR a username (both stored lowercased).
func (s *Service) LoginAdmin(ctx context.Context, email, password string) (domain.TokenPair, domain.Identity, error) {
	ident := strings.ToLower(strings.TrimSpace(email))
	u, err := s.q.GetAdminUserByEmailOrUsername(ctx, sqlcgen.GetAdminUserByEmailOrUsernameParams{
		Email:    ident,
		Username: sql.NullString{String: ident, Valid: ident != ""},
	})
	if err != nil {
		return domain.TokenPair{}, domain.Identity{}, invalidCreds()
	}
	if u.Status != sqlcgen.AdminUsersStatusActive {
		return domain.TokenPair{}, domain.Identity{}, inactiveAccount()
	}
	if !security.VerifyPassword(u.PasswordHash, password) {
		return domain.TokenPair{}, domain.Identity{}, invalidCreds()
	}
	if suspended, err := s.tenantSuspended(ctx, u.StoreID); err != nil {
		return domain.TokenPair{}, domain.Identity{}, err
	} else if suspended {
		return domain.TokenPair{}, domain.Identity{}, suspendedTenant()
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
	if suspended, err := s.tenantSuspended(ctx, st.StoreID); err != nil {
		return domain.TokenPair{}, domain.Identity{}, err
	} else if suspended {
		return domain.TokenPair{}, domain.Identity{}, suspendedTenant()
	}
	if inactive, err := s.packageInactive(ctx, st.StoreID); err != nil {
		return domain.TokenPair{}, domain.Identity{}, err
	} else if inactive {
		return domain.TokenPair{}, domain.Identity{}, packageInactiveErr()
	}

	p := authcontract.Principal{SubjectID: st.ID, StoreID: st.StoreID, Actor: authcontract.ActorStaff, Role: string(st.Role)}
	pair, err := s.issuePair(ctx, p)
	if err != nil {
		return domain.TokenPair{}, domain.Identity{}, err
	}
	return pair, domain.Identity{ID: st.ID, Name: st.Name, Email: st.Email.String, Role: string(st.Role), StoreID: st.StoreID, Actor: authcontract.ActorStaff}, nil
}

// LoginApp validates an external payment API caller's credentials (PLAN.md §10.1.2/§10.1.3) and
// issues a SHORT-LIVED access token — deliberately NOT a token pair: no refresh token is ever
// issued for ActorApp, so this returns just the access token + its TTL. Same generic-error
// discipline as every other login path (don't leak whether an app_id exists at all).
func (s *Service) LoginApp(ctx context.Context, appID, secret string) (accessToken string, expiresIn int64, err error) {
	c, err := s.q.GetPaymentClientForAppLogin(ctx, strings.TrimSpace(appID))
	if err != nil {
		return "", 0, invalidCreds()
	}
	if c.Status != sqlcgen.PaymentClientsStatusActive {
		return "", 0, inactiveAccount()
	}
	if !c.SecretHash.Valid || !security.VerifyPassword(c.SecretHash.String, secret) {
		return "", 0, invalidCreds()
	}
	p := authcontract.Principal{SubjectID: c.ID, Actor: authcontract.ActorApp}
	access, _, err := s.mgr.IssueAccessWithTTL(p, s.mgr.AppTokenTTL())
	if err != nil {
		return "", 0, err
	}
	return access, int64(s.mgr.AppTokenTTL().Seconds()), nil
}

// LoginPlatform validates superadmin (platform operator) credentials and issues a token pair.
// Platform users are NOT scoped to any store — Principal.StoreID stays "".
func (s *Service) LoginPlatform(ctx context.Context, email, password string) (domain.TokenPair, domain.Identity, error) {
	u, err := s.q.GetPlatformUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return domain.TokenPair{}, domain.Identity{}, invalidCreds()
	}
	if u.Status != sqlcgen.PlatformUsersStatusActive {
		return domain.TokenPair{}, domain.Identity{}, inactiveAccount()
	}
	if !security.VerifyPassword(u.PasswordHash, password) {
		return domain.TokenPair{}, domain.Identity{}, invalidCreds()
	}

	p := authcontract.Principal{SubjectID: u.ID, Actor: authcontract.ActorPlatform}
	pair, err := s.issuePair(ctx, p)
	if err != nil {
		return domain.TokenPair{}, domain.Identity{}, err
	}
	return pair, domain.Identity{ID: u.ID, Name: u.Name, Email: u.Email, Actor: authcontract.ActorPlatform}, nil
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
	case authcontract.ActorPlatform:
		u, err := s.q.GetPlatformUserByID(ctx, p.SubjectID)
		if err != nil {
			return domain.Identity{}, invalidSession()
		}
		return domain.Identity{ID: u.ID, Name: u.Name, Email: u.Email, Actor: authcontract.ActorPlatform}, nil
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
		if suspended, err := s.tenantSuspended(ctx, u.StoreID); err != nil {
			return authcontract.Principal{}, err
		} else if suspended {
			return authcontract.Principal{}, suspendedTenant()
		}
		return authcontract.Principal{SubjectID: u.ID, StoreID: u.StoreID, Actor: authcontract.ActorAdmin, Role: string(u.Role)}, nil
	case sqlcgen.RefreshTokensActorStaff:
		st, err := s.q.GetStaffByID(ctx, rt.SubjectID)
		if err != nil || st.Status != sqlcgen.StaffStatusActive {
			return authcontract.Principal{}, invalidSession()
		}
		if suspended, err := s.tenantSuspended(ctx, st.StoreID); err != nil {
			return authcontract.Principal{}, err
		} else if suspended {
			return authcontract.Principal{}, suspendedTenant()
		}
		if inactive, err := s.packageInactive(ctx, st.StoreID); err != nil {
			return authcontract.Principal{}, err
		} else if inactive {
			return authcontract.Principal{}, packageInactiveErr()
		}
		return authcontract.Principal{SubjectID: st.ID, StoreID: st.StoreID, Actor: authcontract.ActorStaff, Role: string(st.Role)}, nil
	case sqlcgen.RefreshTokensActorPlatform:
		u, err := s.q.GetPlatformUserByID(ctx, rt.SubjectID)
		if err != nil || u.Status != sqlcgen.PlatformUsersStatusActive {
			return authcontract.Principal{}, invalidSession()
		}
		return authcontract.Principal{SubjectID: u.ID, Actor: authcontract.ActorPlatform}, nil
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
	switch p.Actor {
	case authcontract.ActorStaff:
		actor = sqlcgen.RefreshTokensActorStaff
	case authcontract.ActorPlatform:
		actor = sqlcgen.RefreshTokensActorPlatform
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
