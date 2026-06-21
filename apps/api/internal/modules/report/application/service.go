// Package application holds the report module's read use cases (analytics).
package application

import (
	"context"
	"time"

	"github.com/elkasir/api/internal/modules/report/infrastructure"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

type Service struct{ repo *infrastructure.Repo }

func NewService(repo *infrastructure.Repo) *Service { return &Service{repo: repo} }

// SummaryDTO ringkasan penjualan untuk dashboard.
type SummaryDTO struct {
	TxCount   int64 `json:"txCount"`
	Revenue   int64 `json:"revenue"`
	CashTotal int64 `json:"cashTotal"`
	QrisTotal int64 `json:"qrisTotal"`
}

// RecentTxDTO satu transaksi terbaru untuk dashboard.
type RecentTxDTO struct {
	ID            string    `json:"id"`
	Code          string    `json:"code"`
	Source        string    `json:"source"`
	PaymentMethod string    `json:"paymentMethod"`
	Total         int64     `json:"total"`
	CreatedAt     time.Time `json:"createdAt"`
}

// DashboardDTO gabungan ringkasan + transaksi terbaru.
type DashboardDTO struct {
	Summary SummaryDTO    `json:"summary"`
	Recent  []RecentTxDTO `json:"recent"`
}

// SalesDayDTO penjualan per hari.
type SalesDayDTO struct {
	Day     string `json:"day"`
	TxCount int64  `json:"txCount"`
	Revenue int64  `json:"revenue"`
}

// TopProductDTO produk terlaris.
type TopProductDTO struct {
	ProductName string `json:"productName"`
	Qty         int64  `json:"qty"`
	Revenue     int64  `json:"revenue"`
}

// CategoryDTO penjualan per kategori.
type CategoryDTO struct {
	Category string `json:"category"`
	Revenue  int64  `json:"revenue"`
	Qty      int64  `json:"qty"`
}

// PaymentDistributionDTO distribusi metode pembayaran.
type PaymentDistributionDTO struct {
	CashTotal int64 `json:"cashTotal"`
	QrisTotal int64 `json:"qrisTotal"`
	CashCount int64 `json:"cashCount"`
	QrisCount int64 `json:"qrisCount"`
}

// StaffDTO performa kasir.
type StaffDTO struct {
	StaffID string `json:"staffId"`
	Name    string `json:"name"`
	TxCount int64  `json:"txCount"`
	Revenue int64  `json:"revenue"`
}

func (s *Service) Dashboard(ctx context.Context, storeID string, from, to time.Time) (DashboardDTO, error) {
	sum, err := s.repo.SalesSummary(ctx, sqlcgen.ReportSalesSummaryParams{
		StoreID: storeID, CreatedAt: from, CreatedAt_2: to,
	})
	if err != nil {
		return DashboardDTO{}, err
	}
	recent, err := s.repo.RecentTransactions(ctx, sqlcgen.ReportRecentTransactionsParams{
		StoreID: storeID, Limit: 10,
	})
	if err != nil {
		return DashboardDTO{}, err
	}
	out := DashboardDTO{
		Summary: SummaryDTO{
			TxCount:   sum.TxCount,
			Revenue:   sum.Revenue,
			CashTotal: sum.CashTotal,
			QrisTotal: sum.QrisTotal,
		},
		Recent: make([]RecentTxDTO, 0, len(recent)),
	}
	for _, t := range recent {
		out.Recent = append(out.Recent, RecentTxDTO{
			ID:            t.ID,
			Code:          t.Code,
			Source:        string(t.Source),
			PaymentMethod: string(t.PaymentMethod),
			Total:         t.Total,
			CreatedAt:     t.CreatedAt,
		})
	}
	return out, nil
}

func (s *Service) Sales(ctx context.Context, storeID string, from, to time.Time) ([]SalesDayDTO, error) {
	rows, err := s.repo.SalesByDay(ctx, sqlcgen.ReportSalesByDayParams{
		StoreID: storeID, CreatedAt: from, CreatedAt_2: to,
	})
	if err != nil {
		return nil, err
	}
	out := make([]SalesDayDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, SalesDayDTO{
			Day:     r.Day.Format("2006-01-02"),
			TxCount: r.TxCount,
			Revenue: r.Revenue,
		})
	}
	return out, nil
}

func (s *Service) TopProducts(ctx context.Context, storeID string, from, to time.Time, limit int) ([]TopProductDTO, error) {
	rows, err := s.repo.TopProducts(ctx, sqlcgen.ReportTopProductsParams{
		StoreID: storeID, CreatedAt: from, CreatedAt_2: to, Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]TopProductDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, TopProductDTO{
			ProductName: r.ProductName,
			Qty:         r.Qty,
			Revenue:     r.Revenue,
		})
	}
	return out, nil
}

func (s *Service) SalesByCategory(ctx context.Context, storeID string, from, to time.Time) ([]CategoryDTO, error) {
	rows, err := s.repo.SalesByCategory(ctx, sqlcgen.ReportSalesByCategoryParams{
		StoreID: storeID, CreatedAt: from, CreatedAt_2: to,
	})
	if err != nil {
		return nil, err
	}
	out := make([]CategoryDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, CategoryDTO{
			Category: r.Category,
			Revenue:  r.Revenue,
			Qty:      r.Qty,
		})
	}
	return out, nil
}

func (s *Service) PaymentDistribution(ctx context.Context, storeID string, from, to time.Time) (PaymentDistributionDTO, error) {
	row, err := s.repo.PaymentDistribution(ctx, sqlcgen.ReportPaymentDistributionParams{
		StoreID: storeID, CreatedAt: from, CreatedAt_2: to,
	})
	if err != nil {
		return PaymentDistributionDTO{}, err
	}
	return PaymentDistributionDTO{
		CashTotal: row.CashTotal,
		QrisTotal: row.QrisTotal,
		CashCount: row.CashCount,
		QrisCount: row.QrisCount,
	}, nil
}

func (s *Service) StaffPerformance(ctx context.Context, storeID string, from, to time.Time) ([]StaffDTO, error) {
	rows, err := s.repo.StaffPerformance(ctx, sqlcgen.ReportStaffPerformanceParams{
		CreatedAt: from, CreatedAt_2: to, StoreID: storeID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]StaffDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, StaffDTO{
			StaffID: r.StaffID,
			Name:    r.Name,
			TxCount: r.TxCount,
			Revenue: r.Revenue,
		})
	}
	return out, nil
}
