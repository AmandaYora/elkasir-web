// Package infrastructure holds the table module's persistence (sqlc + database/sql)
// and its contract implementation.
package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"github.com/elkasir/api/internal/modules/table/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

func toTable(t sqlcgen.DiningTable) domain.Table {
	return domain.Table{
		ID: t.ID, Code: t.Code, Name: t.Name, Area: t.Area,
		Seats: t.Seats, Status: string(t.Status), CreatedAt: t.CreatedAt,
	}
}

func (r *Repo) List(ctx context.Context, f domain.ListFilter) ([]domain.Table, error) {
	rows, err := r.q.ListTables(ctx, f.StoreID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Table, 0, len(rows))
	for _, t := range rows {
		out = append(out, toTable(t))
	}
	return out, nil
}

// Get mengambil satu meja (sql.ErrNoRows bila tak ada).
func (r *Repo) Get(ctx context.Context, storeID, id string) (domain.Table, error) {
	t, err := r.q.GetTable(ctx, sqlcgen.GetTableParams{ID: id, StoreID: storeID})
	if err != nil {
		return domain.Table{}, err
	}
	return toTable(t), nil
}

func (r *Repo) Create(ctx context.Context, storeID, id string, in domain.Input) error {
	return r.q.CreateTable(ctx, sqlcgen.CreateTableParams{
		ID: id, StoreID: storeID,
		Code: strings.TrimSpace(in.Code), Name: nameOf(in), Area: strings.TrimSpace(in.Area),
		Seats: in.Seats, Status: statusOf(in),
	})
}

func (r *Repo) Update(ctx context.Context, storeID, id string, in domain.Input) error {
	return r.q.UpdateTable(ctx, sqlcgen.UpdateTableParams{
		Code: strings.TrimSpace(in.Code), Name: nameOf(in), Area: strings.TrimSpace(in.Area),
		Seats: in.Seats, Status: statusOf(in),
		ID: id, StoreID: storeID,
	})
}

func (r *Repo) Delete(ctx context.Context, storeID, id string) error {
	return r.q.DeleteTable(ctx, sqlcgen.DeleteTableParams{ID: id, StoreID: storeID})
}

func statusOf(in domain.Input) sqlcgen.DiningTablesStatus {
	if in.Status == "inactive" {
		return sqlcgen.DiningTablesStatusInactive
	}
	return sqlcgen.DiningTablesStatusActive
}

func nameOf(in domain.Input) string {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return strings.TrimSpace(in.Code)
	}
	return name
}
