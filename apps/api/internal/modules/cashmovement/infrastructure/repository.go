// Package infrastructure holds the cashmovement module's persistence (sqlc + database/sql).
package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"github.com/elkasir/api/internal/modules/cashmovement/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

func (r *Repo) List(ctx context.Context, storeID string, limit, offset int32) ([]sqlcgen.CashMovement, error) {
	return r.q.ListCashMovements(ctx, sqlcgen.ListCashMovementsParams{StoreID: storeID, Limit: limit, Offset: offset})
}

func (r *Repo) Count(ctx context.Context, storeID string) (int64, error) {
	return r.q.CountCashMovements(ctx, storeID)
}

func (r *Repo) Get(ctx context.Context, storeID, id string) (sqlcgen.CashMovement, error) {
	return r.q.GetCashMovement(ctx, sqlcgen.GetCashMovementParams{ID: id, StoreID: storeID})
}

// CreateInput carries the resolved values needed to persist a cash movement.
type CreateInput struct {
	ID         string
	StoreID    string
	ShiftID    string
	Type       string
	Amount     int64
	Notes      string
	CreatedBy  string
	ApprovedBy string
}

// Create builds the sqlc params and persists the cash movement.
func (r *Repo) Create(ctx context.Context, in CreateInput) error {
	typ, ok := movementType(in.Type)
	if !ok {
		return domain.Input{Type: in.Type}.Validate()
	}
	return r.q.CreateCashMovement(ctx, sqlcgen.CreateCashMovementParams{
		ID: in.ID, StoreID: in.StoreID, ShiftID: nullStr(in.ShiftID), Type: typ, Amount: in.Amount,
		Notes: nullStr(in.Notes), CreatedBy: nullStr(in.CreatedBy), ApprovedBy: nullStr(in.ApprovedBy),
	})
}

func (r *Repo) Settings(ctx context.Context, storeID string) (sqlcgen.Setting, error) {
	return r.q.GetSettingsByStore(ctx, storeID)
}

func movementType(s string) (sqlcgen.CashMovementsType, bool) {
	switch s {
	case domain.TypeCapital:
		return sqlcgen.CashMovementsTypeCapital, true
	case domain.TypeExpense:
		return sqlcgen.CashMovementsTypeExpense, true
	case domain.TypeAdjustment:
		return sqlcgen.CashMovementsTypeAdjustment, true
	default:
		return "", false
	}
}

func nullStr(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
