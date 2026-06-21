// Package application holds the adminuser module's use cases.
package application

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/elkasir/api/internal/modules/adminuser/domain"
	"github.com/elkasir/api/internal/modules/adminuser/infrastructure"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
	"github.com/elkasir/api/internal/platform/security"
)

type Service struct{ repo *infrastructure.Repo }

func NewService(repo *infrastructure.Repo) *Service { return &Service{repo: repo} }

// DTO is the safe API representation of an admin user (without password hash).
type DTO struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Username     string     `json:"username"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	LastActiveAt *time.Time `json:"lastActiveAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

func toDTO(u domain.AdminUser) DTO {
	return DTO{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		Username:     u.Username,
		Role:         u.Role,
		Status:       u.Status,
		LastActiveAt: u.LastActiveAt,
		CreatedAt:    u.CreatedAt,
	}
}

func (s *Service) List(ctx context.Context, f domain.ListFilter) ([]DTO, error) {
	rows, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]DTO, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDTO(u))
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, storeID, uid string) (DTO, error) {
	u, err := s.repo.Get(ctx, storeID, uid)
	if errors.Is(err, sql.ErrNoRows) {
		return DTO{}, httpx.NotFound("Pengguna admin tidak ditemukan.")
	}
	if err != nil {
		return DTO{}, err
	}
	return toDTO(u), nil
}

func (s *Service) Create(ctx context.Context, storeID string, in domain.CreateInput) (DTO, error) {
	if err := in.Validate(); err != nil {
		return DTO{}, err
	}
	email, err := domain.ValidateNameEmail(in.Name, in.Email)
	if err != nil {
		return DTO{}, err
	}
	username, err := domain.ValidateUsername(in.Username)
	if err != nil {
		return DTO{}, err
	}
	hash, err := security.HashPassword(in.Password)
	if err != nil {
		return DTO{}, err
	}
	uid := id.New()
	err = s.repo.Create(ctx, storeID, uid, email, username, hash, in)
	if db.IsDuplicate(err) {
		return DTO{}, httpx.Conflict("Email atau username sudah dipakai.")
	}
	if err != nil {
		return DTO{}, err
	}
	return s.Get(ctx, storeID, uid)
}

func (s *Service) Update(ctx context.Context, storeID, uid string, in domain.UpdateInput) (DTO, error) {
	email, err := domain.ValidateNameEmail(in.Name, in.Email)
	if err != nil {
		return DTO{}, err
	}
	if _, err := s.Get(ctx, storeID, uid); err != nil {
		return DTO{}, err
	}
	err = s.repo.Update(ctx, storeID, uid, email, in)
	if db.IsDuplicate(err) {
		return DTO{}, httpx.Conflict("Email sudah dipakai.")
	}
	if err != nil {
		return DTO{}, err
	}
	return s.Get(ctx, storeID, uid)
}

func (s *Service) ResetPassword(ctx context.Context, storeID, uid, newPassword string) error {
	if _, err := s.Get(ctx, storeID, uid); err != nil {
		return err
	}
	if len(newPassword) < 6 {
		return httpx.Validation("Password minimal 6 karakter.")
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, storeID, uid, hash)
}

func (s *Service) Delete(ctx context.Context, storeID, uid string) error {
	if _, err := s.Get(ctx, storeID, uid); err != nil {
		return err
	}
	return s.repo.Delete(ctx, storeID, uid)
}
