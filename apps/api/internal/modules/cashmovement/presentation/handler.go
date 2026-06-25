// Package presentation holds the cashmovement module's HTTP handlers and routes.
package presentation

import (
	"net/http"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/cashmovement/application"
	"github.com/elkasir/api/internal/modules/cashmovement/domain"
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

// Routes memasang /cash-movements (admin & staf, di balik autentikasi).
func (h *Handler) Routes(r chi.Router) {
	r.Route("/cash-movements", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		// Mutasi kas (modal/biaya/penyesuaian) = supervisor-only di sisi POS (admin web penuh).
		r.Use(authcontract.RequireStaffSupervisorOrAdmin)

		r.Get("/", h.list)
		r.Post("/", h.create)
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page := httpx.PageFromRequest(r, 20, 100)
	f := domain.ListFilter{
		StoreID: authcontract.MustPrincipal(r.Context()).StoreID,
		Limit:   page.Limit,
		Offset:  page.Offset,
	}
	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, page))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in domain.Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	dto, err := h.svc.Create(r.Context(), authcontract.MustPrincipal(r.Context()), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, dto, "Kas berhasil dicatat")
}
