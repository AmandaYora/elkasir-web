// Package application holds the table module's use cases.
package application

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/elkasir/api/internal/modules/table/domain"
	"github.com/elkasir/api/internal/modules/table/infrastructure"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
)

type Service struct{ repo *infrastructure.Repo }

func NewService(repo *infrastructure.Repo) *Service { return &Service{repo: repo} }

// DTO is the API representation of a table (camelCase).
type DTO struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Area      string    `json:"area"`
	Seats     int32     `json:"seats"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(t domain.Table) DTO {
	return DTO{
		ID: t.ID, Code: t.Code, Name: t.Name, Area: t.Area,
		Seats: t.Seats, Status: t.Status, CreatedAt: t.CreatedAt,
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

func (s *Service) Get(ctx context.Context, storeID, tid string) (DTO, error) {
	row, err := s.repo.Get(ctx, storeID, tid)
	if errors.Is(err, sql.ErrNoRows) {
		return DTO{}, httpx.NotFound("Meja tidak ditemukan.")
	}
	if err != nil {
		return DTO{}, err
	}
	return toDTO(row), nil
}

func (s *Service) Create(ctx context.Context, storeID string, in domain.Input) (DTO, error) {
	if err := in.Validate(); err != nil {
		return DTO{}, err
	}
	tid := id.New()
	err := s.repo.Create(ctx, storeID, tid, in)
	if db.IsDuplicate(err) {
		return DTO{}, httpx.Conflict("Kode meja sudah dipakai.")
	}
	if err != nil {
		return DTO{}, err
	}
	return s.Get(ctx, storeID, tid)
}

func (s *Service) Update(ctx context.Context, storeID, tid string, in domain.Input) (DTO, error) {
	if err := in.Validate(); err != nil {
		return DTO{}, err
	}
	if _, err := s.Get(ctx, storeID, tid); err != nil {
		return DTO{}, err
	}
	err := s.repo.Update(ctx, storeID, tid, in)
	if db.IsDuplicate(err) {
		return DTO{}, httpx.Conflict("Kode meja sudah dipakai.")
	}
	if err != nil {
		return DTO{}, err
	}
	return s.Get(ctx, storeID, tid)
}

func (s *Service) Delete(ctx context.Context, storeID, tid string) error {
	if _, err := s.Get(ctx, storeID, tid); err != nil {
		return err
	}
	return s.repo.Delete(ctx, storeID, tid)
}
