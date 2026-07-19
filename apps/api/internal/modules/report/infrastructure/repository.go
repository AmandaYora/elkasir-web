// Package infrastructure holds the report module's read-only persistence (sqlc-backed
// analytics queries). It performs NO writes.
package infrastructure

import (
	"context"
	"database/sql"

	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

func (r *Repo) SalesSummary(ctx context.Context, p sqlcgen.ReportSalesSummaryParams) (sqlcgen.ReportSalesSummaryRow, error) {
	return r.q.ReportSalesSummary(ctx, p)
}

func (r *Repo) SalesByDay(ctx context.Context, p sqlcgen.ReportSalesByDayParams) ([]sqlcgen.ReportSalesByDayRow, error) {
	return r.q.ReportSalesByDay(ctx, p)
}

func (r *Repo) SalesByMonth(ctx context.Context, p sqlcgen.ReportSalesByMonthParams) ([]sqlcgen.ReportSalesByMonthRow, error) {
	return r.q.ReportSalesByMonth(ctx, p)
}

func (r *Repo) TopProducts(ctx context.Context, p sqlcgen.ReportTopProductsParams) ([]sqlcgen.ReportTopProductsRow, error) {
	return r.q.ReportTopProducts(ctx, p)
}

func (r *Repo) SalesByCategory(ctx context.Context, p sqlcgen.ReportSalesByCategoryParams) ([]sqlcgen.ReportSalesByCategoryRow, error) {
	return r.q.ReportSalesByCategory(ctx, p)
}

func (r *Repo) PaymentDistribution(ctx context.Context, p sqlcgen.ReportPaymentDistributionParams) (sqlcgen.ReportPaymentDistributionRow, error) {
	return r.q.ReportPaymentDistribution(ctx, p)
}

func (r *Repo) StaffPerformance(ctx context.Context, p sqlcgen.ReportStaffPerformanceParams) ([]sqlcgen.ReportStaffPerformanceRow, error) {
	return r.q.ReportStaffPerformance(ctx, p)
}

func (r *Repo) RecentTransactions(ctx context.Context, p sqlcgen.ReportRecentTransactionsParams) ([]sqlcgen.ReportRecentTransactionsRow, error) {
	return r.q.ReportRecentTransactions(ctx, p)
}
