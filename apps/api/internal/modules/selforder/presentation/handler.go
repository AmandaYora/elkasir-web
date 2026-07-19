// Package presentation holds the selforder module's HTTP handlers and routes — both the
// PUBLIC (no-auth) customer endpoints and the admin/staff (authenticated) endpoints.
package presentation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/selforder/application"
	"github.com/elkasir/api/internal/platform/httpserver"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
)

// paymentStatusPaid adalah nilai wire (JSON) status pembayaran lunas — dipakai untuk
// menutup stream SSE begitu lunas. Sama dengan yang dikonsumsi frontend.
const paymentStatusPaid = "paid"

type Handler struct {
	svc  *application.Service
	auth authcontract.Authenticator
}

func NewHandler(svc *application.Service, auth authcontract.Authenticator) *Handler {
	return &Handler{svc: svc, auth: auth}
}

func (h *Handler) Routes(r chi.Router) {
	// Publik (tanpa auth) — pelanggan self-order. Rate-limit dasar per-IP.
	// {storeSlug} WAJIB: kode meja cuma unik per-toko (lihat migration 000016), jadi tenant
	// harus di-resolve dari slug, bukan cuma tableCode — lihat tableclient.Client.FindByCode.
	r.Route("/public/order", func(r chi.Router) {
		r.Use(httpserver.RateLimit(60))
		r.Get("/{storeSlug}/{tableCode}", h.menu)
		r.Post("/{storeSlug}/{tableCode}", h.place)
		r.Post("/{storeSlug}/{tableCode}/quote", h.quote)
		r.Get("/status/{selfOrderId}", h.status)
		r.Get("/events/{selfOrderId}", h.events)               // SSE: status pembayaran real-time (pengganti polling)
		r.Post("/{selfOrderId}/simulate-paid", h.simulatePaid) // DEV (gateway nonaktif)
	})

	// Webhook pembayaran TIDAK didaftarkan di sini — Tripay/Midtrans hanya menyediakan SATU
	// callback URL per akun merchant, dibagi lintas semua consumer (selforder, subscription, dan
	// kind=external apps). Route-nya (POST /webhooks/payment) dan dispatch registry-driven-nya
	// (PLAN.md §9.1.5) sekarang milik payment/presentation, yang memanggil
	// Service.ApplyWebhookEvent setelah verifikasi+parse+idempotensi+resolve app_id.

	// Staf/admin — pesanan masuk & tebus barcode.
	r.Route("/self-orders", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		r.Get("/", h.listIncoming) // baca daftar: semua principal terautentikasi

		// Ubah tahap dapur (placed→preparing→completed): staf POS atau admin owner/admin (fallback).
		r.Group(func(r chi.Router) {
			r.Use(authcontract.RequireStaffOrAdmin("owner", "admin"))
			r.Patch("/{id}/status", h.updateStatus)
		})

		// Tebus + terima TUNAI = operasi laci → HANYA staf POS (kasir/supervisor), bukan admin web
		// (dashboard tak punya shift/laci). Web admin hanya memantau pesanan.
		r.Group(func(r chi.Router) {
			r.Use(authcontract.RequireActor(authcontract.ActorStaff))
			r.Get("/redeem/{claimCode}", h.redeem)
			r.Post("/redeem/{claimCode}/checkout", h.redeemCheckout)
		})
	})
}

func (h *Handler) menu(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.Menu(r.Context(), chi.URLParam(r, "storeSlug"), chi.URLParam(r, "tableCode"))
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
	res, err := h.svc.PlaceOrder(r.Context(), chi.URLParam(r, "storeSlug"), chi.URLParam(r, "tableCode"), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, res, "Pesanan berhasil dibuat")
}

func (h *Handler) quote(w http.ResponseWriter, r *http.Request) {
	var in application.PlaceInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	dto, err := h.svc.Quote(r.Context(), chi.URLParam(r, "storeSlug"), chi.URLParam(r, "tableCode"), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.Status(r.Context(), chi.URLParam(r, "selfOrderId"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

// events streams self-order payment status sebagai Server-Sent Events. Layar pelanggan maju
// OTOMATIS begitu callback gateway menandai lunas — tanpa polling. Koneksi dikecualikan dari
// timeout request 30 dtk (lewat header Accept: text/event-stream) dan write deadline-nya
// dinolkan agar bertahan sampai pembayaran. Alurnya: subscribe DULU → kirim snapshot status
// saat ini (menangani kasus sudah-lunas sebelum koneksi dibuka) → teruskan event berikutnya.
func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	soID := chi.URLParam(r, "selfOrderId")

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, httpx.Internal("Streaming tidak didukung."))
		return
	}
	// Lepaskan write deadline (server WriteTimeout) hanya untuk koneksi panjang ini.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	ch, unsubscribe := h.svc.SubscribePayment(soID)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // cegah buffering proxy (nginx)
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	if dto, err := h.svc.Status(r.Context(), soID); err == nil {
		if !writeStatusEvent(w, flusher, dto) || dto.PaymentStatus == paymentStatusPaid {
			return
		}
	}

	ctx := r.Context()
	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-ctx.Done(): // koneksi ditutup klien (atau server shutdown)
			return
		case dto, open := <-ch:
			if !open {
				return
			}
			if !writeStatusEvent(w, flusher, dto) || dto.PaymentStatus == paymentStatusPaid {
				return
			}
		case <-keepalive.C:
			// Komentar SSE menjaga koneksi hidup melewati idle-timeout proxy; gagal tulis = putus.
			if _, err := io.WriteString(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// writeStatusEvent menulis satu event SSE bernama "status" berisi StatusDTO (JSON).
// Mengembalikan false bila penulisan gagal (koneksi putus) agar pemanggil berhenti.
func writeStatusEvent(w http.ResponseWriter, flusher http.Flusher, dto application.StatusDTO) bool {
	payload, err := json.Marshal(dto)
	if err != nil {
		return false
	}
	if _, err := io.WriteString(w, "event: status\ndata: "); err != nil {
		return false
	}
	if _, err := w.Write(payload); err != nil {
		return false
	}
	if _, err := io.WriteString(w, "\n\n"); err != nil {
		return false
	}
	flusher.Flush()
	return true
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
