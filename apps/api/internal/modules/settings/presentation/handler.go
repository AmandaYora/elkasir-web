// Package presentation: HTTP handler modul settings — GET/PATCH /settings (admin) +
// GET /pos/pricing (read-only, admin ATAU staf) untuk POS menghitung layanan & PPN.
package presentation

import (
	"net/http"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/settings/application"
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

// Routes mounts /settings (admin-only; write = owner/admin) + /pos/pricing (any auth).
func (h *Handler) Routes(r chi.Router) {
	r.Route("/settings", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		r.Use(authcontract.RequireActor(authcontract.ActorAdmin))

		r.Get("/", h.get)

		r.Group(func(r chi.Router) {
			r.Use(authcontract.RequireRole("owner", "admin"))
			r.Patch("/", h.update)
		})
	})

	// Konfigurasi harga untuk POS (layanan % + PPN). Read-only, boleh diakses staf kasir
	// (bukan hanya admin) agar app POS menghitung total konsisten dengan server.
	r.With(h.auth.Authenticate).Get("/pos/pricing", h.posPricing)

	// Konfigurasi POS lengkap (pricing + feature flags + ambang kontrol) dalam SATU payload
	// yang ditarik POS saat login & refresh. Server yang memutuskan flag; klien menyembunyikan
	// fitur yang dimatikan. Menggantikan /pos/pricing (yang tetap dipertahankan sbg alias).
	r.With(h.auth.Authenticate).Get("/pos/config", h.posConfig)
}

// PosPricing adalah subset settings yang dibutuhkan POS untuk menghitung breakdown.
type PosPricing struct {
	ServicePercent int32 `json:"servicePercent"`
	TaxPercent     int32 `json:"taxPercent"`
	TaxEnabled     bool  `json:"taxEnabled"`
}

// PosConfig adalah konfigurasi yang ditarik POS (staf/admin) untuk mengatur dirinya:
// harga (layanan/PPN), fitur yang aktif (untuk hide di klien), dan ambang persetujuan.
type PosConfig struct {
	Pricing struct {
		ServicePercent int32 `json:"servicePercent"`
		TaxPercent     int32 `json:"taxPercent"`
		TaxEnabled     bool  `json:"taxEnabled"`
	} `json:"pricing"`
	Features struct {
		Qris         bool `json:"qris"`
		SelfOrder    bool `json:"selfOrder"`
		PayAtCashier bool `json:"payAtCashier"`
	} `json:"features"`
	Thresholds struct {
		MaxDiscountPercent    int32 `json:"maxDiscountPercent"`
		MaxOperationalExpense int64 `json:"maxOperationalExpense"`
		CashVarianceTolerance int64 `json:"cashVarianceTolerance"`
	} `json:"thresholds"`
}

func (h *Handler) posPricing(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.Get(r.Context(), h.storeID(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, PosPricing{
		ServicePercent: dto.ServicePercent,
		TaxPercent:     dto.TaxPercent,
		TaxEnabled:     dto.TaxEnabled,
	})
}

func (h *Handler) posConfig(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.Get(r.Context(), h.storeID(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	var cfg PosConfig
	cfg.Pricing.ServicePercent = dto.ServicePercent
	cfg.Pricing.TaxPercent = dto.TaxPercent
	cfg.Pricing.TaxEnabled = dto.TaxEnabled
	cfg.Features.Qris = dto.FeatureQris
	cfg.Features.SelfOrder = dto.FeatureSelfOrder
	cfg.Features.PayAtCashier = dto.FeaturePayAtCashier
	cfg.Thresholds.MaxDiscountPercent = dto.MaxDiscountPercent
	cfg.Thresholds.MaxOperationalExpense = dto.MaxOperationalExpense
	cfg.Thresholds.CashVarianceTolerance = dto.CashVarianceTolerance
	httpx.OK(w, cfg)
}

func (h *Handler) storeID(r *http.Request) string {
	return authcontract.MustPrincipal(r.Context()).StoreID
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.Get(r.Context(), h.storeID(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in application.Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	dto, err := h.svc.Update(r.Context(), h.storeID(r), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto, "Pengaturan berhasil disimpan")
}
