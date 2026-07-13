// Package presentation holds the subscription module's HTTP handlers and routes — tenant
// (store) billing to the elkasir platform. Admin-only: any admin role may view, only owner
// may check out (moves money). POS staff have no business here — this is not an operational
// POS concern.
package presentation

import (
	"net/http"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/subscription/application"
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

// Routes mounts /subscription (admin-only; read = all admin roles, checkout = owner).
func (h *Handler) Routes(r chi.Router) {
	r.Route("/subscription", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		r.Use(authcontract.RequireActor(authcontract.ActorAdmin))

		r.Get("/plans", h.listPlans)
		r.Get("/", h.current)
		r.Get("/invoices", h.listInvoices)

		r.Group(func(r chi.Router) {
			r.Use(authcontract.RequireRole("owner"))
			r.Post("/checkout", h.checkout)
		})
	})
}

func (h *Handler) listPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.svc.ListPlans(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, plans)
}

func (h *Handler) current(w http.ResponseWriter, r *http.Request) {
	storeID := authcontract.MustPrincipal(r.Context()).StoreID
	dto, err := h.svc.Current(r.Context(), storeID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) checkout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PlanID string `json:"planId"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	storeID := authcontract.MustPrincipal(r.Context()).StoreID
	res, err := h.svc.Checkout(r.Context(), storeID, in.PlanID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, res, "Tagihan langganan berhasil dibuat")
}

func (h *Handler) listInvoices(w http.ResponseWriter, r *http.Request) {
	storeID := authcontract.MustPrincipal(r.Context()).StoreID
	page := httpx.PageFromRequest(r, 20, 100)
	items, total, err := h.svc.ListInvoices(r.Context(), storeID, int32(page.Limit), int32(page.Offset))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, page))
}
