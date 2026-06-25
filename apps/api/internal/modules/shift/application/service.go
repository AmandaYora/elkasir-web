// Package application holds the shift module's use cases.
package application

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	shareddomain "github.com/elkasir/api/internal/domain"
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/shift/domain"
	"github.com/elkasir/api/internal/modules/shift/infrastructure"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
)

type Service struct{ repo *infrastructure.Repo }

func NewService(repo *infrastructure.Repo) *Service { return &Service{repo: repo} }

// DTO is the API representation of a shift (camelCase).
type DTO struct {
	ID                string     `json:"id"`
	StaffID           string     `json:"staffId"`
	Status            string     `json:"status"`
	InitialCash       int64      `json:"initialCash"`
	CashSales         int64      `json:"cashSales"`
	QrisSales         int64      `json:"qrisSales"`
	AdditionalCapital int64      `json:"additionalCapital"`
	Expenses          int64      `json:"expenses"`
	Withdrawals       int64      `json:"withdrawals"`
	Adjustments       int64      `json:"adjustments"`
	DrawerOpenCount   int32      `json:"drawerOpenCount"`
	ExpectedCash      *int64     `json:"expectedCash,omitempty"`
	ActualCash        *int64     `json:"actualCash,omitempty"`
	Variance          *int64     `json:"variance,omitempty"`
	CloseApprovedBy   string     `json:"closeApprovedBy,omitempty"`
	OpenedAt          time.Time  `json:"openedAt"`
	ClosedAt          *time.Time `json:"closedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
}

func toDTO(sh domain.Shift) DTO {
	return DTO{
		ID: sh.ID, StaffID: sh.StaffID, Status: sh.Status,
		InitialCash: sh.InitialCash, CashSales: sh.CashSales, QrisSales: sh.QrisSales,
		AdditionalCapital: sh.AdditionalCapital, Expenses: sh.Expenses,
		Withdrawals: sh.Withdrawals, Adjustments: sh.Adjustments,
		DrawerOpenCount: sh.DrawerOpenCount,
		ExpectedCash:    sh.ExpectedCash, ActualCash: sh.ActualCash, Variance: sh.Variance,
		CloseApprovedBy: sh.CloseApprovedBy,
		OpenedAt:        sh.OpenedAt, ClosedAt: sh.ClosedAt, CreatedAt: sh.CreatedAt,
	}
}

// Open membuka shift baru. Tolak bila masih ada shift terbuka.
func (s *Service) Open(ctx context.Context, p authcontract.Principal, in domain.OpenInput) (DTO, error) {
	if err := in.Validate(); err != nil {
		return DTO{}, err
	}
	storeID := p.StoreID

	_, err := s.repo.GetOpen(ctx, storeID)
	switch {
	case err == nil:
		return DTO{}, httpx.Conflict("Masih ada shift terbuka.")
	case errors.Is(err, sql.ErrNoRows):
		// lanjut buka baru
	default:
		return DTO{}, err
	}

	shiftID := id.New()
	if err := s.repo.Create(ctx, storeID, shiftID, p.SubjectID, in.InitialCash, time.Now().UTC()); err != nil {
		return DTO{}, err
	}

	return s.Get(ctx, storeID, shiftID)
}

// Close menutup shift dengan rekonsiliasi kas (expected vs actual) + kebijakan kontrol.
func (s *Service) Close(ctx context.Context, p authcontract.Principal, shiftID string, in domain.CloseInput) (DTO, error) {
	if err := in.Validate(); err != nil {
		return DTO{}, err
	}
	storeID := p.StoreID

	sh, err := s.repo.Get(ctx, storeID, shiftID)
	if errors.Is(err, sql.ErrNoRows) {
		return DTO{}, httpx.NotFound("Shift tidak ditemukan.")
	}
	if err != nil {
		return DTO{}, err
	}
	if sh.Status != "open" {
		return DTO{}, httpx.Conflict("Shift sudah ditutup.")
	}

	sales, err := s.repo.SalesSummary(ctx, shiftID)
	if err != nil {
		return DTO{}, err
	}
	cm, err := s.repo.CashMovementSummary(ctx, shiftID)
	if err != nil {
		return DTO{}, err
	}

	cash := shareddomain.ShiftCash{
		InitialCash:       sh.InitialCash,
		CashSales:         sales.CashSales,
		AdditionalCapital: cm.Capital,
		Expenses:          cm.Expense,
		Withdrawals:       0,
		Adjustments:       cm.Adjustment,
	}
	expected := cash.ExpectedCash()
	variance := shareddomain.Variance(in.ActualCash, expected)

	policy := s.controlPolicy(ctx, storeID)
	// Selisih kas di atas toleransi butuh persetujuan supervisor — kecuali yang menutup SUDAH
	// supervisor/admin (override otomatis). PIN supervisor diverifikasi di klien; namanya
	// tercatat di closeApprovedBy (audit).
	if policy.VarianceNeedsApproval(variance) && !p.IsSupervisorOrAdmin() && strings.TrimSpace(in.CloseApprovedBy) == "" {
		return DTO{}, httpx.Forbidden("Selisih kas melebihi toleransi; butuh persetujuan supervisor (PIN).")
	}

	n, err := s.repo.Close(ctx, infrastructure.CloseParams{
		StoreID:           storeID,
		ShiftID:           shiftID,
		CashSales:         sales.CashSales,
		QrisSales:         sales.QrisSales,
		AdditionalCapital: cm.Capital,
		Expenses:          cm.Expense,
		Withdrawals:       0,
		Adjustments:       cm.Adjustment,
		DrawerOpenCount:   in.DrawerOpenCount,
		ExpectedCash:      expected,
		ActualCash:        in.ActualCash,
		Variance:          variance,
		CloseApprovedBy:   in.CloseApprovedBy,
		ClosedAt:          time.Now().UTC(),
	})
	if err != nil {
		return DTO{}, err
	}
	if n == 0 {
		return DTO{}, httpx.Conflict("Shift sudah ditutup.")
	}

	return s.Get(ctx, storeID, shiftID)
}

func (s *Service) Get(ctx context.Context, storeID, sid string) (DTO, error) {
	sh, err := s.repo.Get(ctx, storeID, sid)
	if errors.Is(err, sql.ErrNoRows) {
		return DTO{}, httpx.NotFound("Shift tidak ditemukan.")
	}
	if err != nil {
		return DTO{}, err
	}
	return toDTO(sh), nil
}

func (s *Service) List(ctx context.Context, f domain.ListFilter) ([]DTO, int64, error) {
	rows, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx, f.StoreID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]DTO, 0, len(rows))
	for _, sh := range rows {
		out = append(out, toDTO(sh))
	}
	return out, total, nil
}

// Current mengembalikan shift terbuka, atau (nil, nil) bila tidak ada.
func (s *Service) Current(ctx context.Context, storeID string) (*DTO, error) {
	sh, err := s.repo.GetOpen(ctx, storeID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d := toDTO(sh)
	return &d, nil
}

func (s *Service) controlPolicy(ctx context.Context, storeID string) shareddomain.ControlPolicy {
	st, err := s.repo.Settings(ctx, storeID)
	if err != nil || !st.Found {
		// default aman bila settings belum ada
		return shareddomain.ControlPolicy{MaxDiscountPercent: 10, MaxOperationalExpense: 200000, CashVarianceTolerance: 5000}
	}
	return shareddomain.ControlPolicy{
		MaxDiscountPercent:    st.MaxDiscountPercent,
		MaxOperationalExpense: st.MaxOperationalExpense,
		CashVarianceTolerance: st.CashVarianceTolerance,
	}
}
