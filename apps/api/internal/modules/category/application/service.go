// Package application holds the category module's use cases.
package application

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/elkasir/api/internal/modules/category/domain"
	"github.com/elkasir/api/internal/modules/category/infrastructure"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
)

type Service struct{ repo *infrastructure.Repo }

func NewService(repo *infrastructure.Repo) *Service { return &Service{repo: repo} }

// DTO is the API representation of a category (camelCase).
type DTO struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	SortOrder    int32     `json:"sortOrder"`
	ProductCount int64     `json:"productCount"`
	CreatedAt    time.Time `json:"createdAt"`
}

func toDTO(c domain.Category) DTO {
	return DTO{
		ID: c.ID, Name: c.Name, SortOrder: c.SortOrder,
		ProductCount: c.ProductCount, CreatedAt: c.CreatedAt,
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

func (s *Service) Get(ctx context.Context, storeID, cid string) (DTO, error) {
	row, err := s.repo.Get(ctx, storeID, cid)
	if errors.Is(err, sql.ErrNoRows) {
		return DTO{}, httpx.NotFound("Kategori tidak ditemukan.")
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
	cid := id.New()
	err := s.repo.Create(ctx, storeID, cid, in)
	if db.IsDuplicate(err) {
		return DTO{}, httpx.Conflict("Nama kategori sudah dipakai.")
	}
	if err != nil {
		return DTO{}, err
	}
	return s.Get(ctx, storeID, cid)
}

func (s *Service) Update(ctx context.Context, storeID, cid string, in domain.Input) (DTO, error) {
	if err := in.Validate(); err != nil {
		return DTO{}, err
	}
	if _, err := s.Get(ctx, storeID, cid); err != nil {
		return DTO{}, err
	}
	err := s.repo.Update(ctx, storeID, cid, in)
	if db.IsDuplicate(err) {
		return DTO{}, httpx.Conflict("Nama kategori sudah dipakai.")
	}
	if err != nil {
		return DTO{}, err
	}
	return s.Get(ctx, storeID, cid)
}

func (s *Service) Delete(ctx context.Context, storeID, cid string) error {
	if _, err := s.Get(ctx, storeID, cid); err != nil {
		return err
	}
	return s.repo.Delete(ctx, storeID, cid)
}
