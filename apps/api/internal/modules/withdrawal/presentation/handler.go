// Package presentation holds the withdrawal module's HTTP handlers and routes.
package presentation

import (
	"net/http"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/withdrawal/application"
	"github.com/elkasir/api/internal/modules/withdrawal/domain"
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

// Routes mounts /withdrawals (admin-only; read = all admins, request = owner).
func (h *Handler) Routes(r chi.Router) {
	r.Route("/withdrawals", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		r.Use(authcontract.RequireActor(authcontract.ActorAdmin))

		r.Get("/", h.list)
		r.Get("/balance", h.balance)

		r.Group(func(r chi.Router) {
			r.Use(authcontract.RequireRole("owner"))
			r.Post("/", h.create)
		})
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p := authcontract.MustPrincipal(r.Context())
	page := httpx.PageFromRequest(r, 20, 100)
	f := domain.ListFilter{StoreID: p.StoreID, Limit: int32(page.Limit), Offset: int32(page.Offset)}
	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, page))
}

func (h *Handler) balance(w http.ResponseWriter, r *http.Request) {
	storeID := authcontract.MustPrincipal(r.Context()).StoreID
	balance, err := h.svc.Balance(r.Context(), storeID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, struct {
		AvailableBalance int64 `json:"availableBalance"`
	}{AvailableBalance: balance})
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
	httpx.Created(w, dto, "Pengajuan pencairan berhasil dibuat")
}
