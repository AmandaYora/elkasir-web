// Package presentation holds the adminuser module's HTTP handlers and routes.
package presentation

import (
	"net/http"

	"github.com/elkasir/api/internal/modules/adminuser/application"
	"github.com/elkasir/api/internal/modules/adminuser/domain"
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
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

// Routes mounts /admin-users (admin-only; writes = owner/admin).
func (h *Handler) Routes(r chi.Router) {
	r.Route("/admin-users", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		r.Use(authcontract.RequireActor(authcontract.ActorAdmin))

		r.Get("/", h.list)
		r.Get("/{id}", h.get)

		r.Group(func(r chi.Router) {
			r.Use(authcontract.RequireRole("owner", "admin"))
			r.Post("/", h.create)
			r.Put("/{id}", h.update)
			r.Post("/{id}/reset-password", h.resetPassword)
			r.Delete("/{id}", h.delete)
		})
	})
}

func (h *Handler) storeID(r *http.Request) string {
	return authcontract.MustPrincipal(r.Context()).StoreID
}

// guardOwner enforces that only an owner may grant the `owner` role or modify an existing
// owner account — closing an admin→owner privilege escalation. The web UI also hides these,
// so a normal admin never reaches this; it only blocks a direct-API bypass.
func (h *Handler) guardOwner(r *http.Request, targetID, assignRole string) error {
	caller := authcontract.MustPrincipal(r.Context())
	if caller.Role == "owner" {
		return nil
	}
	if assignRole == "owner" {
		return httpx.Forbidden("Hanya owner yang dapat menetapkan role owner.")
	}
	if targetID != "" {
		existing, err := h.svc.Get(r.Context(), caller.StoreID, targetID)
		if err != nil {
			return err
		}
		if existing.Role == "owner" {
			return httpx.Forbidden("Hanya owner yang dapat mengubah akun owner.")
		}
	}
	return nil
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
	if err := h.guardOwner(r, "", in.Role); err != nil {
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
	if err := h.guardOwner(r, chi.URLParam(r, "id"), in.Role); err != nil {
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
	if err := h.guardOwner(r, chi.URLParam(r, "id"), ""); err != nil {
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
	if err := h.guardOwner(r, chi.URLParam(r, "id"), ""); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.Delete(r.Context(), h.storeID(r), chi.URLParam(r, "id")); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}
