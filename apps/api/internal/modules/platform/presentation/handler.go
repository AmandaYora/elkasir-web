// Package presentation holds the platform module's HTTP handlers and routes — superadmin only
// (ActorPlatform). Every route here is deliberately cross-tenant (tenant lifecycle, revenue,
// plan catalog); nothing about a single tenant's operational data is exposed.
package presentation

import (
	"net/http"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/modules/platform/application"
	"github.com/elkasir/api/internal/modules/platform/domain"
	platformuserclient "github.com/elkasir/api/internal/modules/platformuser/contracts"
	subscriptionclient "github.com/elkasir/api/internal/modules/subscription/contracts"
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

// Routes mounts /platform — every endpoint here requires ActorPlatform (superadmin).
func (h *Handler) Routes(r chi.Router) {
	r.Route("/platform", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		r.Use(authcontract.RequireActor(authcontract.ActorPlatform))

		r.Get("/tenants", h.listTenants)
		r.Post("/tenants", h.createTenant)
		r.Patch("/tenants/{id}/status", h.setTenantStatus)
		r.Get("/tenants/revenue", h.tenantsRevenue)

		r.Get("/revenue", h.revenue)

		r.Get("/plans", h.listPlans)
		r.Post("/plans", h.createPlan)
		r.Patch("/plans/{id}", h.updatePlan)

		r.Get("/withdrawals", h.listActiveWithdrawals)
		r.Patch("/withdrawals/{id}/claim", h.claimWithdrawal)
		r.Patch("/withdrawals/{id}/success", h.completeWithdrawal)
		r.Patch("/withdrawals/{id}/reject", h.rejectWithdrawal)
		r.Get("/withdrawals/history", h.withdrawalHistory)

		r.Get("/users", h.listPlatformUsers)
		r.Post("/users", h.createPlatformUser)
		r.Patch("/users/{id}/status", h.setPlatformUserStatus)
		r.Patch("/users/{id}/reset-password", h.resetPlatformUserPassword)

		// Payment gateway config + app registry (PLAN.md §9.1.10, Part 2).
		r.Get("/payment-config", h.getPaymentConfig)
		r.Put("/payment-config", h.updatePaymentConfig)
		r.Get("/payment-clients", h.listPaymentApps)
		r.Post("/payment-clients", h.createPaymentApp)
		r.Patch("/payment-clients/{id}/reset-secret", h.resetPaymentAppSecret)
		r.Patch("/payment-clients/{id}/status", h.setPaymentAppStatus)
	})
}

func (h *Handler) listTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.svc.ListTenants(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, tenants)
}

func (h *Handler) createTenant(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateTenantInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	tenant, err := h.svc.CreateTenant(r.Context(), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, tenant, "Tenant berhasil dibuat")
}

func (h *Handler) setTenantStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	tenant, err := h.svc.SetTenantStatus(r.Context(), chi.URLParam(r, "id"), in.Status)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, tenant)
}

func (h *Handler) revenue(w http.ResponseWriter, r *http.Request) {
	summary, err := h.svc.Revenue(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, summary)
}

func (h *Handler) listPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.svc.ListPlans(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, plans)
}

func (h *Handler) createPlan(w http.ResponseWriter, r *http.Request) {
	var in subscriptionclient.PlanInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	plan, err := h.svc.CreatePlan(r.Context(), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, plan, "Paket berhasil dibuat")
}

func (h *Handler) updatePlan(w http.ResponseWriter, r *http.Request) {
	var in subscriptionclient.PlanInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	plan, err := h.svc.UpdatePlan(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, plan)
}

func (h *Handler) tenantsRevenue(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.TenantsRevenue(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, rows)
}

func (h *Handler) listActiveWithdrawals(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.ListActiveWithdrawals(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, rows)
}

func (h *Handler) claimWithdrawal(w http.ResponseWriter, r *http.Request) {
	p := authcontract.MustPrincipal(r.Context())
	if err := h.svc.ClaimWithdrawal(r.Context(), chi.URLParam(r, "id"), p.SubjectID); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) completeWithdrawal(w http.ResponseWriter, r *http.Request) {
	p := authcontract.MustPrincipal(r.Context())
	if err := h.svc.CompleteWithdrawal(r.Context(), chi.URLParam(r, "id"), p.SubjectID); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) rejectWithdrawal(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Reason string `json:"reason"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	p := authcontract.MustPrincipal(r.Context())
	if err := h.svc.RejectWithdrawal(r.Context(), chi.URLParam(r, "id"), p.SubjectID, in.Reason); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) withdrawalHistory(w http.ResponseWriter, r *http.Request) {
	page := httpx.PageFromRequest(r, 20, 100)
	rows, total, err := h.svc.ListWithdrawalHistory(r.Context(), int32(page.Limit), int32(page.Offset))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(rows, total, page))
}

func (h *Handler) listPlatformUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.ListPlatformUsers(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, rows)
}

func (h *Handler) createPlatformUser(w http.ResponseWriter, r *http.Request) {
	var in platformuserclient.CreateInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	u, err := h.svc.CreatePlatformUser(r.Context(), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, u, "User platform berhasil dibuat")
}

func (h *Handler) setPlatformUserStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	p := authcontract.MustPrincipal(r.Context())
	if err := h.svc.SetPlatformUserStatus(r.Context(), p.SubjectID, chi.URLParam(r, "id"), in.Status); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) resetPlatformUserPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.ResetPlatformUserPassword(r.Context(), chi.URLParam(r, "id"), in.Password); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

// ── Payment gateway config + app registry (PLAN.md §9.1.10, Part 2) ─────────────────────────

func (h *Handler) getPaymentConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.svc.GetPaymentConfig(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, cfg)
}

// updatePaymentConfigRequest mirrors paymentclient.UpdateGatewayConfigInput's write-only-secret
// shape (§9.1.2): a secret field OMITTED from the request body stays nil (unchanged); a field
// present in the body — even "" — is applied. json.Unmarshal into a *string does exactly this.
type updatePaymentConfigRequest struct {
	Provider           string  `json:"provider"`
	Sandbox            bool    `json:"sandbox"`
	TripayAPIKey       *string `json:"tripayApiKey"`
	TripayPrivateKey   *string `json:"tripayPrivateKey"`
	TripayMerchantCode *string `json:"tripayMerchantCode"`
	TripayMethod       string  `json:"tripayMethod"`
	MidtransServerKey  *string `json:"midtransServerKey"`
}

func (h *Handler) updatePaymentConfig(w http.ResponseWriter, r *http.Request) {
	var in updatePaymentConfigRequest
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	cfg, err := h.svc.UpdatePaymentConfig(r.Context(), paymentclient.UpdateGatewayConfigInput{
		Provider: in.Provider, Sandbox: in.Sandbox, TripayAPIKey: in.TripayAPIKey,
		TripayPrivateKey: in.TripayPrivateKey, TripayMerchantCode: in.TripayMerchantCode,
		TripayMethod: in.TripayMethod, MidtransServerKey: in.MidtransServerKey,
	})
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, cfg, "Konfigurasi pembayaran berhasil disimpan")
}

func (h *Handler) listPaymentApps(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.ListPaymentApps(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, rows)
}

func (h *Handler) createPaymentApp(w http.ResponseWriter, r *http.Request) {
	var in paymentclient.CreateAppInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	result, err := h.svc.CreatePaymentApp(r.Context(), in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, result, "Aplikasi berhasil didaftarkan")
}

func (h *Handler) resetPaymentAppSecret(w http.ResponseWriter, r *http.Request) {
	secret, err := h.svc.ResetPaymentAppSecret(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, map[string]string{"secret": secret})
}

func (h *Handler) setPaymentAppStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.SetPaymentAppStatus(r.Context(), chi.URLParam(r, "id"), in.Status); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}
