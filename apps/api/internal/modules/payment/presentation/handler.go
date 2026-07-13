// Package presentation holds the payment module's HTTP routes: the webhook endpoint registered
// with the active gateway's dashboard, and (Part 3, §10) the external payment API for
// registered `kind=external` apps. Superadmin-facing config/registry management is NOT here
// (§9.1.10) — that lives in `platform`'s own presentation, calling through paymentclient.Client
// like any other cross-module contract consumer. ActorApp, by contrast, is genuinely payment's
// own caller type with no other natural home (§10.1.8) — hence these routes live here directly.
package presentation

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/ratelimit"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	client     paymentclient.Client
	dispatcher paymentclient.Dispatcher
	auth       authcontract.Authenticator
	// limiter is keyed by the calling app's row ID (ActorApp's SubjectID) — 60/min per §10.1.11.
	// A single shared instance for all three /external/payments/* routes (not per-route), since
	// the limit is meant to bound one app's TOTAL call volume, not each route independently.
	limiter *ratelimit.Limiter
}

func NewHandler(client paymentclient.Client, auth authcontract.Authenticator) *Handler {
	return &Handler{
		client:     client,
		dispatcher: client.(paymentclient.Dispatcher),
		auth:       auth,
		limiter:    ratelimit.New(60, time.Minute),
	}
}

// Routes mounts the webhook endpoint (unchanged since Part 2) and the external payment API
// (Part 3, §10.1.8) — all three external routes gated RequireActor(ActorApp).
func (h *Handler) Routes(r chi.Router) {
	r.Post("/webhooks/payment", h.webhook)

	r.Route("/external/payments", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		r.Use(authcontract.RequireActor(authcontract.ActorApp))
		r.Use(h.rateLimitByApp)

		r.Post("/charges", h.createCharge)
		r.Get("/charges/{orderRef}/status", h.chargeStatus)
		r.Get("/channels", h.listChannels)
	})
}

// rateLimitByApp implements the per-app_id limit (§10.1.11) — must run AFTER Authenticate, since
// it keys off the resolved principal, not the raw request.
func (h *Handler) rateLimitByApp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := authcontract.MustPrincipal(r.Context())
		if !h.limiter.Allow(p.SubjectID) {
			httpx.Error(w, httpx.RateLimited("Terlalu banyak permintaan. Coba lagi sebentar lagi."))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		httpx.Error(w, httpx.BadRequest("Body webhook tidak terbaca."))
		return
	}
	if !h.client.VerifyWebhook(r.Header, body) {
		httpx.Error(w, httpx.Unauthorized("Callback pembayaran tidak terverifikasi."))
		return
	}
	ev, err := h.client.ParseWebhook(body)
	if errors.Is(err, paymentclient.ErrInvalidPayload) {
		httpx.Error(w, httpx.BadRequest("Payload webhook tidak valid."))
		return
	}
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if ev.EventID == "" {
		httpx.OK(w, map[string]string{"received": "ok"}) // tak ada identitas event → abaikan
		return
	}

	ctx := r.Context()
	seen, err := h.client.WebhookSeen(ctx, ev.EventID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if !seen {
		if err := h.dispatcher.Dispatch(ctx, ev); err != nil {
			httpx.Error(w, err) // gagal sementara → biarkan provider retry (belum ditandai seen)
			return
		}
		if err := h.client.MarkWebhookSeen(ctx, ev.EventID); err != nil {
			httpx.Error(w, err)
			return
		}
	}
	httpx.OK(w, map[string]string{"received": "ok"})
}

// ── External payment API (Part 3, §10) ────────────────────────────────────────────────────────

type createChargeRequest struct {
	OrderRef       string `json:"orderRef"`
	Amount         int64  `json:"amount"`
	Channel        string `json:"channel"` // "qris" (default) | "virtual_account"
	ChannelOptions struct {
		BankCode string `json:"bankCode"`
	} `json:"channelOptions"`
}

func (h *Handler) createCharge(w http.ResponseWriter, r *http.Request) {
	var req createChargeRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	if strings.TrimSpace(req.OrderRef) == "" || req.Amount <= 0 {
		httpx.Error(w, httpx.Validation("orderRef dan amount (>0) wajib diisi."))
		return
	}
	channel := paymentclient.Channel(req.Channel)
	if channel == "" {
		channel = paymentclient.ChannelQRIS
	}

	ctx := r.Context()
	p := authcontract.MustPrincipal(ctx)
	app, err := h.dispatcher.ResolveApp(ctx, p.SubjectID)
	if err != nil {
		httpx.Error(w, httpx.Unauthorized("Aplikasi tidak dikenali."))
		return
	}

	// storeID kosong (§10.1.5) — pemanggil eksternal sejati tidak terikat toko Elkasir mana pun;
	// gateway sendiri tidak pernah memakai storeID (lihat CreateChannelCharge).
	charge, err := h.client.CreateChannelCharge(ctx, app.AppID, "", req.OrderRef, req.Amount, channel,
		paymentclient.ChannelOptions{BankCode: req.ChannelOptions.BankCode})
	if err != nil {
		if db.IsDuplicate(err) {
			httpx.Error(w, httpx.Conflict(
				"orderRef sudah pernah dipakai. Gunakan GET /external/payments/charges/{orderRef}/status untuk memeriksa status, jangan membuat ulang."))
			return
		}
		httpx.Error(w, httpx.Internal("Gagal membuat tagihan: "+err.Error()))
		return
	}
	httpx.Created(w, charge, "Tagihan berhasil dibuat")
}

func (h *Handler) chargeStatus(w http.ResponseWriter, r *http.Request) {
	orderRef := chi.URLParam(r, "orderRef")
	ctx := r.Context()
	p := authcontract.MustPrincipal(ctx)

	ownerAppID, providerRef, err := h.dispatcher.ResolveCharge(ctx, orderRef)
	if err != nil {
		httpx.Error(w, httpx.NotFound("orderRef tidak ditemukan."))
		return
	}
	app, err := h.dispatcher.ResolveApp(ctx, p.SubjectID)
	if err != nil || app.AppID != ownerAppID {
		// Sengaja 404, bukan 403 — jangan konfirmasi ke pemanggil bahwa orderRef ini ADA tapi
		// milik aplikasi lain (§10.2 EB2: tidak boleh bisa menebak-nebak charge aplikasi lain).
		httpx.Error(w, httpx.NotFound("orderRef tidak ditemukan."))
		return
	}
	if providerRef == "" {
		httpx.Error(w, httpx.Internal("Tagihan ini belum memiliki referensi gateway (mode simulasi?)."))
		return
	}
	status, err := h.client.CheckStatus(ctx, providerRef)
	if err != nil {
		httpx.Error(w, httpx.Internal("Gagal memeriksa status: "+err.Error()))
		return
	}
	httpx.OK(w, status)
}

func (h *Handler) listChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := h.client.ListChannels(r.Context())
	if err != nil {
		httpx.Error(w, httpx.Internal("Gagal mengambil daftar kanal: "+err.Error()))
		return
	}
	httpx.OK(w, channels)
}
