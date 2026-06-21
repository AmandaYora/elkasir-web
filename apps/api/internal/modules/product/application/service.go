// Package application holds the product module's use cases.
package application

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/elkasir/api/internal/modules/product/domain"
	"github.com/elkasir/api/internal/modules/product/infrastructure"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
)

type Service struct{ repo *infrastructure.Repo }

func NewService(repo *infrastructure.Repo) *Service { return &Service{repo: repo} }

// DTO is the API representation of a product (camelCase).
type DTO struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Sku        string    `json:"sku"`
	CategoryID string    `json:"categoryId"`
	Category   string    `json:"category"`
	Price      int64     `json:"price"`
	Cost       int64     `json:"cost"`
	Stock      int32     `json:"stock"`
	Status     string    `json:"status"`
	ImageURL   string    `json:"imageUrl"`
	CreatedAt  time.Time `json:"createdAt"`
}

func toDTO(p domain.Product) DTO {
	return DTO{
		ID: p.ID, Name: p.Name, Sku: p.Sku, CategoryID: p.CategoryID, Category: p.Category,
		Price: p.Price, Cost: p.Cost, Stock: p.Stock, Status: p.Status, ImageURL: p.ImageURL, CreatedAt: p.CreatedAt,
	}
}

func (s *Service) List(ctx context.Context, f domain.ListFilter) ([]DTO, int64, error) {
	rows, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]DTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toDTO(r))
	}
	return out, total, nil
}

func (s *Service) Get(ctx context.Context, storeID, pid string) (DTO, error) {
	row, err := s.repo.Get(ctx, storeID, pid)
	if errors.Is(err, sql.ErrNoRows) {
		return DTO{}, httpx.NotFound("Produk tidak ditemukan.")
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
	pid := id.New()
	err := s.repo.Create(ctx, storeID, pid, in)
	if db.IsDuplicate(err) {
		return DTO{}, httpx.Conflict("SKU sudah dipakai produk lain.")
	}
	if db.IsForeignKey(err) {
		return DTO{}, httpx.Validation("Kategori tidak valid.")
	}
	if err != nil {
		return DTO{}, err
	}
	return s.Get(ctx, storeID, pid)
}

func (s *Service) Update(ctx context.Context, storeID, pid string, in domain.Input) (DTO, error) {
	if err := in.Validate(); err != nil {
		return DTO{}, err
	}
	if _, err := s.Get(ctx, storeID, pid); err != nil {
		return DTO{}, err
	}
	err := s.repo.Update(ctx, storeID, pid, in)
	if db.IsDuplicate(err) {
		return DTO{}, httpx.Conflict("SKU sudah dipakai produk lain.")
	}
	if db.IsForeignKey(err) {
		return DTO{}, httpx.Validation("Kategori tidak valid.")
	}
	if err != nil {
		return DTO{}, err
	}
	return s.Get(ctx, storeID, pid)
}

func (s *Service) Delete(ctx context.Context, storeID, pid string) error {
	if _, err := s.Get(ctx, storeID, pid); err != nil {
		return err
	}
	return s.repo.Delete(ctx, storeID, pid)
}

// AdjustStock increments/decrements stock (delta may be negative).
func (s *Service) AdjustStock(ctx context.Context, storeID, pid string, delta int64) (DTO, error) {
	n, err := s.repo.AdjustStock(ctx, storeID, pid, delta)
	if err != nil {
		return DTO{}, err
	}
	if n == 0 {
		return DTO{}, httpx.NotFound("Produk tidak ditemukan.")
	}
	return s.Get(ctx, storeID, pid)
}
