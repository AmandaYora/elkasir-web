// Package presentation holds the shift module's HTTP handlers and routes.
package presentation

import (
	"net/http"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/shift/application"
	"github.com/elkasir/api/internal/modules/shift/domain"
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

// Routes: GET list/detail/current (admin & staf); buka/tutup shift = staf POS.
func (h *Handler) Routes(r chi.Router) {
	r.Route("/shifts", func(r chi.Router) {
		r.Use(h.auth.Authenticate)

		r.Get("/", h.list)
		r.Get("/current", h.current)
		r.Get("/{id}", h.get)

		r.Group(func(r chi.Router) {
			r.Use(authcontract.RequireActor(authcontract.ActorStaff))
			r.Post("/", h.open)
			r.Post("/{id}/close", h.close)
		})
	})
}

func (h *Handler) storeID(r *http.Request) string {
	return authcontract.MustPrincipal(r.Context()).StoreID
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page := httpx.PageFromRequest(r, 20, 100)
	f := domain.ListFilter{StoreID: h.storeID(r), Limit: page.Limit, Offset: page.Offset}
	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, page))
}

func (h *Handler) current(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.Current(r.Context(), h.storeID(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if dto == nil {
		httpx.NoContent(w)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.Get(r.Context(), h.storeID(r), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) open(w http.ResponseWriter, r *http.Request) {
	var in domain.OpenInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	dto, err := h.svc.Open(r.Context(), authcontract.MustPrincipal(r.Context()), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, dto, "Shift berhasil dibuka")
}

func (h *Handler) close(w http.ResponseWriter, r *http.Request) {
	var in domain.CloseInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	dto, err := h.svc.Close(r.Context(), authcontract.MustPrincipal(r.Context()), chi.URLParam(r, "id"), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto, "Shift berhasil ditutup")
}
