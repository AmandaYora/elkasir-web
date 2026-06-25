// Package presentation holds the report module's HTTP handlers and routes (read-only).
package presentation

import (
	"net/http"
	"time"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/report/application"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc  *application.Service
	auth authcontract.Authenticator
}

func NewHandler(svc *application.Service, auth authcontract.Authenticator) *Handler {
	return &Handler{svc: svc, auth: auth}
}

// Routes memasang /reports (admin & staf; read-only analitik).
func (h *Handler) Routes(r chi.Router) {
	r.Route("/reports", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		// Laporan/analitik = supervisor-only di sisi POS (admin web tetap penuh).
		r.Use(authcontract.RequireStaffSupervisorOrAdmin)

		r.Get("/dashboard", h.dashboard)
		r.Get("/sales", h.sales)
		r.Get("/top-products", h.topProducts)
		r.Get("/sales-by-category", h.salesByCategory)
		r.Get("/payment-distribution", h.paymentDistribution)
		r.Get("/staff-performance", h.staffPerformance)
	})
}

func (h *Handler) storeID(r *http.Request) string {
	return authcontract.MustPrincipal(r.Context()).StoreID
}

// timeRange membaca query param `from`/`to` (RFC3339 atau "2006-01-02").
// Default: 30 hari terakhir hingga besok bila tidak diisi.
func timeRange(r *http.Request) (time.Time, time.Time) {
	now := time.Now()
	to := parseQueryTime(httpx.QueryStr(r, "to", ""), now.AddDate(0, 0, 1))
	from := parseQueryTime(httpx.QueryStr(r, "from", ""), now.AddDate(0, 0, -30))
	return from, to
}

func parseQueryTime(s string, def time.Time) time.Time {
	if s == "" {
		return def
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return def
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	from, to := timeRange(r)
	res, err := h.svc.Dashboard(r.Context(), h.storeID(r), from, to)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, res)
}

func (h *Handler) sales(w http.ResponseWriter, r *http.Request) {
	from, to := timeRange(r)
	res, err := h.svc.Sales(r.Context(), h.storeID(r), from, to)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, res)
}

func (h *Handler) topProducts(w http.ResponseWriter, r *http.Request) {
	from, to := timeRange(r)
	limit := httpx.QueryInt(r, "limit", 10)
	res, err := h.svc.TopProducts(r.Context(), h.storeID(r), from, to, limit)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, res)
}

func (h *Handler) salesByCategory(w http.ResponseWriter, r *http.Request) {
	from, to := timeRange(r)
	res, err := h.svc.SalesByCategory(r.Context(), h.storeID(r), from, to)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, res)
}

func (h *Handler) paymentDistribution(w http.ResponseWriter, r *http.Request) {
	from, to := timeRange(r)
	res, err := h.svc.PaymentDistribution(r.Context(), h.storeID(r), from, to)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, res)
}

func (h *Handler) staffPerformance(w http.ResponseWriter, r *http.Request) {
	from, to := timeRange(r)
	res, err := h.svc.StaffPerformance(r.Context(), h.storeID(r), from, to)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, res)
}
