// Package application holds the staff module's use cases.
package application

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/elkasir/api/internal/modules/staff/domain"
	"github.com/elkasir/api/internal/modules/staff/infrastructure"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
	"github.com/elkasir/api/internal/platform/security"
)

type Service struct{ repo *infrastructure.Repo }

func NewService(repo *infrastructure.Repo) *Service { return &Service{repo: repo} }

// DTO is the API representation of a staff member (without password hash).
type DTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(s domain.Staff) DTO {
	return DTO{
		ID:        s.ID,
		Name:      s.Name,
		Username:  s.Username,
		Email:     s.Email,
		Role:      s.Role,
		Status:    s.Status,
		CreatedAt: s.CreatedAt,
	}
}

func (s *Service) List(ctx context.Context, f domain.ListFilter) ([]DTO, error) {
	rows, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]DTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toDTO(r))
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, storeID, sid string) (DTO, error) {
	row, err := s.repo.Get(ctx, storeID, sid)
	if errors.Is(err, sql.ErrNoRows) {
		return DTO{}, httpx.NotFound("Staf tidak ditemukan.")
	}
	if err != nil {
		return DTO{}, err
	}
	return toDTO(row), nil
}

func (s *Service) Create(ctx context.Context, storeID string, in domain.CreateInput) (DTO, error) {
	if err := in.Validate(); err != nil {
		return DTO{}, err
	}
	hash, err := security.HashPassword(in.Password)
	if err != nil {
		return DTO{}, err
	}
	sid := id.New()
	err = s.repo.Create(ctx, storeID, sid, hash, in)
	if db.IsDuplicate(err) {
		return DTO{}, httpx.Conflict("Username sudah dipakai.")
	}
	if err != nil {
		return DTO{}, err
	}
	return s.Get(ctx, storeID, sid)
}

func (s *Service) Update(ctx context.Context, storeID, sid string, in domain.UpdateInput) (DTO, error) {
	if err := in.Validate(); err != nil {
		return DTO{}, err
	}
	if _, err := s.Get(ctx, storeID, sid); err != nil {
		return DTO{}, err
	}
	err := s.repo.Update(ctx, storeID, sid, in)
	if db.IsDuplicate(err) {
		return DTO{}, httpx.Conflict("Username sudah dipakai.")
	}
	if err != nil {
		return DTO{}, err
	}
	return s.Get(ctx, storeID, sid)
}

func (s *Service) ResetPassword(ctx context.Context, storeID, sid, newPassword string) error {
	if len(newPassword) < 6 {
		return httpx.Validation("Password minimal 6 karakter.")
	}
	if _, err := s.Get(ctx, storeID, sid); err != nil {
		return err
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, storeID, sid, hash)
}

func (s *Service) Delete(ctx context.Context, storeID, sid string) error {
	if _, err := s.Get(ctx, storeID, sid); err != nil {
		return err
	}
	return s.repo.Delete(ctx, storeID, sid)
}
