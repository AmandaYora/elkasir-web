// Package application holds the platformuser module's use cases (implementing
// platformuserclient.Client — superadmin account management, PLAN.md §2.8/§2.9).
package application

import (
	"context"
	"database/sql"
	"errors"

	platformuserclient "github.com/elkasir/api/internal/modules/platformuser/contracts"
	"github.com/elkasir/api/internal/modules/platformuser/domain"
	"github.com/elkasir/api/internal/modules/platformuser/infrastructure"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
	"github.com/elkasir/api/internal/platform/security"
)

type Service struct{ repo *infrastructure.Repo }

func NewService(repo *infrastructure.Repo) *Service { return &Service{repo: repo} }

var _ platformuserclient.Client = (*Service)(nil)

func toClient(u domain.PlatformUser) platformuserclient.PlatformUser {
	return platformuserclient.PlatformUser{ID: u.ID, Name: u.Name, Email: u.Email, Status: u.Status, CreatedAt: u.CreatedAt}
}

// List implements platformuserclient.Client.
func (s *Service) List(ctx context.Context) ([]platformuserclient.PlatformUser, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]platformuserclient.PlatformUser, 0, len(rows))
	for _, u := range rows {
		out = append(out, toClient(u))
	}
	return out, nil
}

// Create implements platformuserclient.Client. Never hard-deletes, no role tiers (§2.9) — every
// created account is a full, equal superadmin.
func (s *Service) Create(ctx context.Context, in platformuserclient.CreateInput) (platformuserclient.PlatformUser, error) {
	domainIn := domain.CreateInput{Name: in.Name, Email: in.Email, Password: in.Password}
	if err := domainIn.Validate(); err != nil {
		return platformuserclient.PlatformUser{}, err
	}
	hash, err := security.HashPassword(in.Password)
	if err != nil {
		return platformuserclient.PlatformUser{}, err
	}
	uid := id.New()
	if err := s.repo.Create(ctx, uid, in.Name, in.Email, hash); err != nil {
		if db.IsDuplicate(err) {
			return platformuserclient.PlatformUser{}, httpx.Conflict("Email sudah dipakai.")
		}
		return platformuserclient.PlatformUser{}, err
	}
	u, err := s.repo.Get(ctx, uid)
	if err != nil {
		return platformuserclient.PlatformUser{}, err
	}
	return toClient(u), nil
}

// SetStatus implements platformuserclient.Client — a superadmin cannot deactivate their own
// account (§2.9); never hard-deletes (deactivate only).
func (s *Service) SetStatus(ctx context.Context, actingUserID, targetID, status string) error {
	if status != "active" && status != "inactive" {
		return httpx.Validation("Status harus 'active' atau 'inactive'.")
	}
	if actingUserID == targetID && status == "inactive" {
		return httpx.Forbidden("Anda tidak dapat menonaktifkan akun Anda sendiri.")
	}
	n, err := s.repo.SetStatus(ctx, targetID, status)
	if err != nil {
		return err
	}
	if n == 0 {
		return httpx.NotFound("User platform tidak ditemukan.")
	}
	return nil
}

// ResetPassword implements platformuserclient.Client.
func (s *Service) ResetPassword(ctx context.Context, id, newPassword string) error {
	if len(newPassword) < 6 {
		return httpx.Validation("Password minimal 6 karakter.")
	}
	if _, err := s.repo.Get(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.NotFound("User platform tidak ditemukan.")
		}
		return err
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, id, hash)
}
