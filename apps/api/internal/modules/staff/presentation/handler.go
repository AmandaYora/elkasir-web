// Package presentation holds the staff module's HTTP handlers and routes.
package presentation

import (
	"net/http"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/staff/application"
	"github.com/elkasir/api/internal/modules/staff/domain"
	"github.com/elkasir/api/internal/platform/httpserver"
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

// Routes mounts /staff (admin-only; writes = owner/admin).
func (h *Handler) Routes(r chi.Router) {
	r.Route("/staff", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		r.Use(authcontract.RequireActor(authcontract.ActorAdmin))

		r.Get("/", h.list)
		r.Get("/{id}", h.get)

		r.Group(func(r chi.Router) {
			r.Use(authcontract.RequireRole("owner", "admin"))
			r.Post("/", h.create)
			r.Put("/{id}", h.update)
			r.Post("/{id}/reset-password", h.resetPassword)
			r.Put("/{id}/pin", h.setPin) // set/hapus PIN supervisor
			r.Delete("/{id}", h.delete)
		})
	})

	// Verifikasi PIN supervisor untuk persetujuan in-place dari POS (staf). Rate-limited
	// terhadap brute-force PIN; mengembalikan identitas supervisor pencocok.
	r.Route("/pos/approvals", func(r chi.Router) {
		r.Use(httpserver.RateLimit(20))
		r.Use(h.auth.Authenticate)
		r.Use(authcontract.RequireActor(authcontract.ActorStaff))
		r.Post("/verify-pin", h.verifyPin)
	})
}

func (h *Handler) storeID(r *http.Request) string {
	return authcontract.MustPrincipal(r.Context()).StoreID
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context(), domain.ListFilter{StoreID: h.storeID(r)})
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, int64(len(items)), httpx.Page{Limit: len(items), Offset: 0}))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.Get(r.Context(), h.storeID(r), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	dto, err := h.svc.Create(r.Context(), h.storeID(r), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, dto)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in domain.UpdateInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	dto, err := h.svc.Update(r.Context(), h.storeID(r), chi.URLParam(r, "id"), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), h.storeID(r), chi.URLParam(r, "id"), body.Password); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), h.storeID(r), chi.URLParam(r, "id")); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

// setPin menyetel/menghapus PIN supervisor (admin owner/admin). Body: {"pin":"1234"} atau {"pin":""}.
func (h *Handler) setPin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Pin string `json:"pin"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.SetPin(r.Context(), h.storeID(r), chi.URLParam(r, "id"), body.Pin); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

// verifyPin mencocokkan PIN supervisor (staf POS, rate-limited) → identitas supervisor penyetuju.
func (h *Handler) verifyPin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Pin string `json:"pin"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	ref, err := h.svc.VerifySupervisorPIN(r.Context(), h.storeID(r), body.Pin)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, ref)
}
