// Package presentation holds the product module's HTTP handlers and routes.
package presentation

import (
	"net/http"

	"github.com/elkasir/api/internal/modules/product/application"
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/product/domain"
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

// Routes mounts /products. Reads (catalog) are available to any authenticated
// principal — the admin web AND the POS staff app both need the menu to operate.
// Writes stay admin-only (owner/admin); the catalog is managed from the web.
func (h *Handler) Routes(r chi.Router) {
	r.Route("/products", func(r chi.Router) {
		r.Use(h.auth.Authenticate)

		r.Get("/", h.list)
		r.Get("/{id}", h.get)

		r.Group(func(r chi.Router) {
			r.Use(authcontract.RequireActor(authcontract.ActorAdmin))
			r.Use(authcontract.RequireRole("owner", "admin"))
			r.Post("/", h.create)
			r.Put("/{id}", h.update)
			r.Delete("/{id}", h.delete)
			r.Post("/{id}/adjust-stock", h.adjustStock)
		})
	})
}

func (h *Handler) storeID(r *http.Request) string {
	return authcontract.MustPrincipal(r.Context()).StoreID
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page := httpx.PageFromRequest(r, 20, 100)
	f := domain.ListFilter{
		StoreID:    h.storeID(r),
		Status:     httpx.QueryStr(r, "status", ""),
		CategoryID: httpx.QueryStr(r, "categoryId", ""),
		Search:     httpx.QueryStr(r, "search", ""),
		Limit:      page.Limit,
		Offset:     page.Offset,
	}
	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, page))
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
	var in domain.Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	dto, err := h.svc.Create(r.Context(), h.storeID(r), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, dto, "Produk berhasil dibuat")
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in domain.Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	dto, err := h.svc.Update(r.Context(), h.storeID(r), chi.URLParam(r, "id"), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto, "Produk berhasil diperbarui")
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), h.storeID(r), chi.URLParam(r, "id")); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) adjustStock(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Delta int64 `json:"delta"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	dto, err := h.svc.AdjustStock(r.Context(), h.storeID(r), chi.URLParam(r, "id"), body.Delta)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto, "Stok berhasil disesuaikan")
}
