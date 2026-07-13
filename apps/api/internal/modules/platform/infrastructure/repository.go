// Package infrastructure holds the platform module's persistence. It reads/writes the
// tenant-lifecycle columns on `stores` (slug, status) — a shared-kernel exception, the same
// pattern already used by `settings` for the profile columns (see knowledge/MODULE_MAP.md).
// Tenant CREATION itself goes through bootstrap.ProvisionTenant (called from the application
// layer), not this repo — provisioning a store+settings+owner atomically is infra-level
// bootstrapping, not a single-table repo operation.
package infrastructure

import (
	"context"
	"database/sql"

	"github.com/elkasir/api/internal/modules/platform/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

func (r *Repo) List(ctx context.Context) ([]domain.Tenant, error) {
	rows, err := r.q.ListStores(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Tenant, 0, len(rows))
	for _, s := range rows {
		out = append(out, domain.Tenant{ID: s.ID, Name: s.Name, Slug: s.Slug, Status: string(s.Status), CreatedAt: s.CreatedAt})
	}
	return out, nil
}

func (r *Repo) Get(ctx context.Context, storeID string) (domain.Tenant, error) {
	s, err := r.q.GetStoreByID(ctx, storeID)
	if err != nil {
		return domain.Tenant{}, err
	}
	return domain.Tenant{ID: s.ID, Name: s.Name, Slug: s.Slug, Status: string(s.Status), CreatedAt: s.CreatedAt}, nil
}

// SetStatus returns rows-affected — 0 means the tenant id doesn't exist.
func (r *Repo) SetStatus(ctx context.Context, storeID, status string) (int64, error) {
	return r.q.UpdateStoreStatus(ctx, sqlcgen.UpdateStoreStatusParams{
		Status: sqlcgen.StoresStatus(status), ID: storeID,
	})
}
