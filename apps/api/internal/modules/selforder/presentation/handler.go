// Package presentation holds the selforder module's HTTP handlers and routes — both the
// PUBLIC (no-auth) customer endpoints and the admin/staff (authenticated) endpoints.
package presentation

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/selforder/application"
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

func (h *Handler) Routes(r chi.Router) {
	// Publik (tanpa auth) — pelanggan self-order. Rate-limit dasar per-IP.
	r.Route("/public/order", func(r chi.Router) {
		r.Use(httpserver.RateLimit(60))
		r.Get("/{tableCode}", h.menu)
		r.Post("/{tableCode}", h.place)
		r.Get("/status/{selfOrderId}", h.status)
		r.Post("/{selfOrderId}/simulate-paid", h.simulatePaid) // DEV (tanpa Xendit)
	})

	// Webhook Xendit (verifikasi token di handler).
	r.Post("/webhooks/xendit", h.webhook)

	// Staf/admin — pesanan masuk & tebus barcode.
	r.Route("/self-orders", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		r.Get("/", h.listIncoming)
		r.Patch("/{id}/status", h.updateStatus)
		r.Get("/redeem/{claimCode}", h.redeem)
		r.Post("/redeem/{claimCode}/checkout", h.redeemCheckout)
	})
}

func (h *Handler) menu(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.Menu(r.Context(), chi.URLParam(r, "tableCode"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) place(w http.ResponseWriter, r *http.Request) {
	var in application.PlaceInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	res, err := h.svc.PlaceOrder(r.Context(), chi.URLParam(r, "tableCode"), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, res, "Pesanan berhasil dibuat")
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.Status(r.Context(), chi.URLParam(r, "selfOrderId"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) simulatePaid(w http.ResponseWriter, r *http.Request) {
	// Hanya tersedia di mode dev (gateway belum dikonfigurasi). Produksi → pakai webhook.
	if h.svc.PaymentEnabled() {
		httpx.Error(w, httpx.NotFound("Endpoint tidak tersedia."))
		return
	}
	if err := h.svc.SimulatePaid(r.Context(), chi.URLParam(r, "selfOrderId")); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, map[string]string{"status": "paid"})
}

func (h *Handler) webhook(w http.ResponseWriter, r *http.Request) {
	// Header & body diteruskan apa adanya; verifikasi (skema provider) ada di module payment.
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		httpx.Error(w, httpx.BadRequest("Body webhook tidak terbaca."))
		return
	}
	if err := h.svc.HandleWebhook(r.Context(), r.Header, body); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, map[string]string{"received": "ok"})
}

func (h *Handler) listIncoming(w http.ResponseWriter, r *http.Request) {
	page := httpx.PageFromRequest(r, 50, 200)
	storeID := authcontract.MustPrincipal(r.Context()).StoreID
	items, total, err := h.svc.ListIncoming(r.Context(), storeID, httpx.QueryStr(r, "status", ""), page.Limit, page.Offset)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, page))
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	storeID := authcontract.MustPrincipal(r.Context()).StoreID
	dto, err := h.svc.UpdateStatus(r.Context(), storeID, chi.URLParam(r, "id"), body.Status)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) redeem(w http.ResponseWriter, r *http.Request) {
	storeID := authcontract.MustPrincipal(r.Context()).StoreID
	dto, err := h.svc.Redeem(r.Context(), storeID, chi.URLParam(r, "claimCode"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) redeemCheckout(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	sum := sha256.Sum256(body)
	reqHash := hex.EncodeToString(sum[:])
	idemKey := r.Header.Get("Idempotency-Key")

	res, err := h.svc.RedeemCheckout(r.Context(), authcontract.MustPrincipal(r.Context()), chi.URLParam(r, "claimCode"), idemKey, reqHash)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, res)
}
