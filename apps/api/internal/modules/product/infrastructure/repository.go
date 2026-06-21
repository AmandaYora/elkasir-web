// Package infrastructure holds the product module's persistence (sqlc + database/sql)
// and its contract implementation.
package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"github.com/elkasir/api/internal/modules/product/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

const selectRow = `SELECT p.id, COALESCE(p.category_id, ''), COALESCE(c.name, ''), COALESCE(p.sku, ''),
	p.name, p.price, p.cost, p.stock, p.status, COALESCE(p.image_url, ''), p.created_at
	FROM products p LEFT JOIN product_categories c ON c.id = p.category_id `

// List returns filtered products + total (for pagination). The dynamic query (optional
// filters) is hand-written; sqlc handles the static operations.
func (r *Repo) List(ctx context.Context, f domain.ListFilter) ([]domain.Product, int64, error) {
	var where strings.Builder
	where.WriteString("WHERE p.store_id = ?")
	args := []any{f.StoreID}

	if f.Status != "" {
		where.WriteString(" AND p.status = ?")
		args = append(args, f.Status)
	}
	if f.CategoryID != "" {
		where.WriteString(" AND p.category_id = ?")
		args = append(args, f.CategoryID)
	}
	if f.Search != "" {
		where.WriteString(" AND (p.name LIKE ? OR p.sku LIKE ?)")
		like := "%" + f.Search + "%"
		args = append(args, like, like)
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM products p "+where.String(), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataSQL := selectRow + where.String() + " ORDER BY p.created_at DESC LIMIT ? OFFSET ?"
	rows, err := r.db.QueryContext(ctx, dataSQL, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.Product, 0, f.Limit)
	for rows.Next() {
		var x domain.Product
		if err := rows.Scan(&x.ID, &x.CategoryID, &x.Category, &x.Sku, &x.Name,
			&x.Price, &x.Cost, &x.Stock, &x.Status, &x.ImageURL, &x.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, x)
	}
	return out, total, rows.Err()
}

// Get fetches one product + category name (sql.ErrNoRows when absent).
func (r *Repo) Get(ctx context.Context, storeID, id string) (domain.Product, error) {
	var x domain.Product
	err := r.db.QueryRowContext(ctx, selectRow+"WHERE p.id = ? AND p.store_id = ? LIMIT 1", id, storeID).
		Scan(&x.ID, &x.CategoryID, &x.Category, &x.Sku, &x.Name, &x.Price, &x.Cost, &x.Stock, &x.Status, &x.ImageURL, &x.CreatedAt)
	return x, err
}

func (r *Repo) Create(ctx context.Context, storeID, id string, in domain.Input) error {
	return r.q.CreateProduct(ctx, sqlcgen.CreateProductParams{
		ID: id, StoreID: storeID,
		CategoryID: nullStr(in.CategoryID), Sku: nullStr(in.Sku),
		Name: strings.TrimSpace(in.Name), Price: in.Price, Cost: in.Cost,
		Stock: in.Stock, Status: statusOf(in.Status), ImageUrl: nullStr(in.ImageURL),
	})
}

func (r *Repo) Update(ctx context.Context, storeID, id string, in domain.Input) error {
	return r.q.UpdateProduct(ctx, sqlcgen.UpdateProductParams{
		CategoryID: nullStr(in.CategoryID), Sku: nullStr(in.Sku),
		Name: strings.TrimSpace(in.Name), Price: in.Price, Cost: in.Cost,
		Stock: in.Stock, Status: statusOf(in.Status), ImageUrl: nullStr(in.ImageURL),
		ID: id, StoreID: storeID,
	})
}

func (r *Repo) Delete(ctx context.Context, storeID, id string) error {
	return r.q.DeleteProduct(ctx, sqlcgen.DeleteProductParams{ID: id, StoreID: storeID})
}

func (r *Repo) AdjustStock(ctx context.Context, storeID, id string, delta int64) (int64, error) {
	return r.q.AdjustProductStock(ctx, sqlcgen.AdjustProductStockParams{Stock: int32(delta), ID: id, StoreID: storeID})
}

func statusOf(s string) sqlcgen.ProductsStatus {
	if s == "inactive" {
		return sqlcgen.ProductsStatusInactive
	}
	return sqlcgen.ProductsStatusActive
}

func nullStr(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
