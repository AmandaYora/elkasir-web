// Package infrastructure holds the shift module's persistence (sqlc + database/sql)
// and its contract implementation. Penutupan memakai perhitungan domain
// (ExpectedCash/Variance) sebagai satu sumber kebenaran.
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/elkasir/api/internal/modules/shift/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

// SalesSummary adalah ringkasan penjualan sebuah shift.
type SalesSummary struct {
	CashSales int64
	QrisSales int64
}

// CashMovementSummary adalah ringkasan pergerakan kas sebuah shift.
type CashMovementSummary struct {
	Capital    int64
	Expense    int64
	Adjustment int64
}

// ControlSettings adalah ambang kontrol toko (untuk ControlPolicy).
type ControlSettings struct {
	MaxDiscountPercent    int64
	MaxOperationalExpense int64
	CashVarianceTolerance int64
	Found                 bool
}

// CloseParams adalah parameter penutupan shift yang sudah terhitung.
type CloseParams struct {
	StoreID           string
	ShiftID           string
	CashSales         int64
	QrisSales         int64
	AdditionalCapital int64
	Expenses          int64
	Withdrawals       int64
	Adjustments       int64
	DrawerOpenCount   int32
	ExpectedCash      int64
	ActualCash        int64
	Variance          int64
	CloseApprovedBy   string
	ClosedAt          time.Time
}

func toShift(sh sqlcgen.Shift) domain.Shift {
	return domain.Shift{
		ID: sh.ID, StaffID: sh.StaffID, Status: string(sh.Status),
		InitialCash: sh.InitialCash, CashSales: sh.CashSales, QrisSales: sh.QrisSales,
		AdditionalCapital: sh.AdditionalCapital, Expenses: sh.Expenses,
		Withdrawals: sh.Withdrawals, Adjustments: sh.Adjustments,
		DrawerOpenCount: sh.DrawerOpenCount,
		ExpectedCash:    nullInt(sh.ExpectedCash), ActualCash: nullInt(sh.ActualCash), Variance: nullInt(sh.Variance),
		CloseApprovedBy: sh.CloseApprovedBy.String,
		OpenedAt:        sh.OpenedAt, ClosedAt: nullTime(sh.ClosedAt), CreatedAt: sh.CreatedAt,
	}
}

// GetOpen mengembalikan shift terbuka (sql.ErrNoRows bila tak ada).
func (r *Repo) GetOpen(ctx context.Context, storeID string) (domain.Shift, error) {
	sh, err := r.q.GetOpenShift(ctx, storeID)
	if err != nil {
		return domain.Shift{}, err
	}
	return toShift(sh), nil
}

func (r *Repo) Get(ctx context.Context, storeID, id string) (domain.Shift, error) {
	sh, err := r.q.GetShift(ctx, sqlcgen.GetShiftParams{ID: id, StoreID: storeID})
	if err != nil {
		return domain.Shift{}, err
	}
	return toShift(sh), nil
}

func (r *Repo) List(ctx context.Context, f domain.ListFilter) ([]domain.Shift, error) {
	rows, err := r.q.ListShifts(ctx, sqlcgen.ListShiftsParams{StoreID: f.StoreID, Limit: int32(f.Limit), Offset: int32(f.Offset)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Shift, 0, len(rows))
	for _, sh := range rows {
		out = append(out, toShift(sh))
	}
	return out, nil
}

func (r *Repo) Count(ctx context.Context, storeID string) (int64, error) {
	return r.q.CountShifts(ctx, storeID)
}

func (r *Repo) Create(ctx context.Context, storeID, id, staffID string, initialCash int64, openedAt time.Time) error {
	return r.q.CreateShift(ctx, sqlcgen.CreateShiftParams{
		ID: id, StoreID: storeID, StaffID: staffID,
		InitialCash: initialCash, OpenedAt: openedAt,
	})
}

// SalesSummary mengembalikan ringkasan penjualan untuk shift tertentu.
func (r *Repo) SalesSummary(ctx context.Context, shiftID string) (SalesSummary, error) {
	row, err := r.q.ShiftSalesSummary(ctx, sql.NullString{String: shiftID, Valid: true})
	if err != nil {
		return SalesSummary{}, err
	}
	return SalesSummary{CashSales: row.CashSales, QrisSales: row.QrisSales}, nil
}

// CashMovementSummary mengembalikan ringkasan pergerakan kas untuk shift tertentu.
func (r *Repo) CashMovementSummary(ctx context.Context, shiftID string) (CashMovementSummary, error) {
	row, err := r.q.ShiftCashMovementSummary(ctx, sql.NullString{String: shiftID, Valid: true})
	if err != nil {
		return CashMovementSummary{}, err
	}
	return CashMovementSummary{Capital: row.Capital, Expense: row.Expense, Adjustment: row.Adjustment}, nil
}

// Settings mengembalikan ambang kontrol toko. Found=false bila settings belum ada.
func (r *Repo) Settings(ctx context.Context, storeID string) (ControlSettings, error) {
	st, err := r.q.GetSettingsByStore(ctx, storeID)
	if errors.Is(err, sql.ErrNoRows) {
		return ControlSettings{Found: false}, nil
	}
	if err != nil {
		return ControlSettings{}, err
	}
	return ControlSettings{
		MaxDiscountPercent:    int64(st.MaxDiscountPercent),
		MaxOperationalExpense: st.MaxOperationalExpense,
		CashVarianceTolerance: st.CashVarianceTolerance,
		Found:                 true,
	}, nil
}

// Close menutup shift dengan parameter yang sudah terhitung. Mengembalikan jumlah baris terdampak.
func (r *Repo) Close(ctx context.Context, p CloseParams) (int64, error) {
	return r.q.CloseShift(ctx, sqlcgen.CloseShiftParams{
		CashSales:         p.CashSales,
		QrisSales:         p.QrisSales,
		AdditionalCapital: p.AdditionalCapital,
		Expenses:          p.Expenses,
		Withdrawals:       p.Withdrawals,
		Adjustments:       p.Adjustments,
		DrawerOpenCount:   p.DrawerOpenCount,
		ExpectedCash:      sql.NullInt64{Int64: p.ExpectedCash, Valid: true},
		ActualCash:        sql.NullInt64{Int64: p.ActualCash, Valid: true},
		Variance:          sql.NullInt64{Int64: p.Variance, Valid: true},
		CloseApprovedBy:   nullStr(p.CloseApprovedBy),
		ClosedAt:          sql.NullTime{Time: p.ClosedAt, Valid: true},
		ID:                p.ShiftID,
		StoreID:           p.StoreID,
	})
}

func nullStr(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func nullTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
