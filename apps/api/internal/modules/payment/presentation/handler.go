// Package presentation holds the payment module's HTTP routes: the webhook endpoint registered
// with the active gateway's dashboard (Tripay/Midtrans, for selforder), and the inbound webhook
// receiver for ElProof's relay (subscription billing — see infrastructure/elproof.go). Part 3's
// external-facing payment API (a separate SaaS product calling THROUGH Elkasir's own wallet) was
// removed — Elkasir is now a client of a dedicated external product (ElProof) instead of also
// being a provider to others; see PLAN.md §11.
package presentation

import (
	"errors"
	"io"
	"net/http"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	client     paymentclient.Client
	dispatcher paymentclient.Dispatcher
	auth       authcontract.Authenticator
}

func NewHandler(client paymentclient.Client, auth authcontract.Authenticator) *Handler {
	return &Handler{
		client:     client,
		dispatcher: client.(paymentclient.Dispatcher),
		auth:       auth,
	}
}

// Routes mounts both inbound gateway webhooks: Elkasir's own Tripay/Midtrans wallet
// (/webhooks/payment, unchanged since Part 2) and ElProof's relay for subscription billing
// (/webhooks/payment/elproof — this is the callbackUrl registered with ElProof for
// `Elkasir-Billing`).
func (h *Handler) Routes(r chi.Router) {
	r.Post("/webhooks/payment", h.webhook)
	r.Post("/webhooks/payment/elproof", h.elproofWebhook)
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
	provider := h.client.ActiveProviderName()
	seen, err := h.client.WebhookSeen(ctx, provider, ev.EventID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if !seen {
		if err := h.dispatcher.Dispatch(ctx, ev); err != nil {
			httpx.Error(w, err) // gagal sementara → biarkan provider retry (belum ditandai seen)
			return
		}
		if err := h.client.MarkWebhookSeen(ctx, provider, ev.EventID); err != nil {
			httpx.Error(w, err)
			return
		}
	}
	httpx.OK(w, map[string]string{"received": "ok"})
}

// elproofWebhook receives the relay ElProof sends when a subscription-billing charge (appID
// paymentclient.AppSubscribe) gets paid. Distinct route from /webhooks/payment because ElProof's
// signature scheme, payload shape, and secret are entirely different from Elkasir's own
// Tripay/Midtrans callback (see infrastructure/elproof.go) — it is never "the active gateway",
// so it cannot share that path's dispatch. Reuses the SAME Dispatch() as /webhooks/payment once
// the payload is normalized into paymentclient.WebhookEvent: payment_charge_apps already has the
// order_ref→AppSubscribe index written at checkout time (CreateChannelCharge), so no change to
// the dispatcher itself was needed.
func (h *Handler) elproofWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		httpx.Error(w, httpx.BadRequest("Body webhook tidak terbaca."))
		return
	}
	if !h.client.VerifyElProofWebhook(r.Header, body) {
		httpx.Error(w, httpx.Unauthorized("Callback ElProof tidak terverifikasi."))
		return
	}
	ev, err := h.client.ParseElProofWebhook(body)
	if errors.Is(err, paymentclient.ErrInvalidPayload) {
		httpx.Error(w, httpx.BadRequest("Payload webhook tidak valid."))
		return
	}
	if err != nil {
		httpx.Error(w, err)
		return
	}

	ctx := r.Context()
	const elproofProvider = "elproof"
	seen, err := h.client.WebhookSeen(ctx, elproofProvider, ev.EventID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if !seen {
		if err := h.dispatcher.Dispatch(ctx, ev); err != nil {
			httpx.Error(w, err)
			return
		}
		if err := h.client.MarkWebhookSeen(ctx, elproofProvider, ev.EventID); err != nil {
			httpx.Error(w, err)
			return
		}
	}
	httpx.OK(w, map[string]string{"received": "ok"})
}
