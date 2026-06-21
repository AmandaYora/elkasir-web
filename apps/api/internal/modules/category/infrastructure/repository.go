// Package infrastructure holds the category module's persistence (sqlc + database/sql).
package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"github.com/elkasir/api/internal/modules/category/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

// List returns the store's categories with their product counts.
func (r *Repo) List(ctx context.Context, f domain.ListFilter) ([]domain.Category, error) {
	rows, err := r.q.ListCategories(ctx, f.StoreID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Category, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Category{
			ID: row.ID, Name: row.Name, SortOrder: row.SortOrder,
			ProductCount: row.ProductCount, CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

// Get fetches one category (sql.ErrNoRows when absent).
func (r *Repo) Get(ctx context.Context, storeID, id string) (domain.Category, error) {
	c, err := r.q.GetCategory(ctx, sqlcgen.GetCategoryParams{ID: id, StoreID: storeID})
	if err != nil {
		return domain.Category{}, err
	}
	return domain.Category{
		ID: c.ID, Name: c.Name, SortOrder: c.SortOrder,
		ProductCount: 0, CreatedAt: c.CreatedAt,
	}, nil
}

func (r *Repo) Create(ctx context.Context, storeID, id string, in domain.Input) error {
	return r.q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ID: id, StoreID: storeID,
		Name: strings.TrimSpace(in.Name), SortOrder: in.SortOrder,
	})
}

func (r *Repo) Update(ctx context.Context, storeID, id string, in domain.Input) error {
	return r.q.UpdateCategory(ctx, sqlcgen.UpdateCategoryParams{
		Name: strings.TrimSpace(in.Name), SortOrder: in.SortOrder,
		ID: id, StoreID: storeID,
	})
}

func (r *Repo) Delete(ctx context.Context, storeID, id string) error {
	return r.q.DeleteCategory(ctx, sqlcgen.DeleteCategoryParams{ID: id, StoreID: storeID})
}
