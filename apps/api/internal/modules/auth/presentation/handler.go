// Package presentation holds the auth module's HTTP handlers and routes.
package presentation

import (
	"net/http"

	"github.com/elkasir/api/internal/modules/auth/application"
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/auth/domain"
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

// Routes mounts the auth endpoints.
func (h *Handler) Routes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		// Anti brute-force: batasi percobaan login & refresh token per IP (klien asli dari
		// RealIP/nginx). Kredensial salah berulang akan ditahan 429.
		r.Group(func(r chi.Router) {
			r.Use(httpserver.RateLimit(20))
			r.Post("/admin/login", h.adminLogin)
			r.Post("/staff/login", h.staffLogin)
			r.Post("/platform/login", h.platformLogin)
			r.Post("/refresh", h.refresh)
		})
		r.Post("/logout", h.logout)

		r.Group(func(r chi.Router) {
			r.Use(h.auth.Authenticate)
			r.Get("/me", h.me)
		})
	})
}

type adminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type staffLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type userDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email,omitempty"`
	Role    string `json:"role"`
	StoreID string `json:"storeId"`
	Actor   string `json:"actor"`
}

type loginResponse struct {
	AccessToken  string  `json:"accessToken"`
	RefreshToken string  `json:"refreshToken"`
	ExpiresIn    int64   `json:"expiresIn"`
	User         userDTO `json:"user"`
}

type tokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

func toUserDTO(i domain.Identity) userDTO {
	return userDTO{ID: i.ID, Name: i.Name, Email: i.Email, Role: i.Role, StoreID: i.StoreID, Actor: string(i.Actor)}
}

func (h *Handler) adminLogin(w http.ResponseWriter, r *http.Request) {
	var req adminLoginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	if req.Email == "" || req.Password == "" {
		httpx.Error(w, httpx.Validation("Email dan password wajib diisi."))
		return
	}
	pair, identity, err := h.svc.LoginAdmin(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, loginResponse{
		AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: pair.ExpiresIn,
		User: toUserDTO(identity),
	}, "Login berhasil")
}

func (h *Handler) staffLogin(w http.ResponseWriter, r *http.Request) {
	var req staffLoginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	if req.Username == "" || req.Password == "" {
		httpx.Error(w, httpx.Validation("Username dan password wajib diisi."))
		return
	}
	pair, identity, err := h.svc.LoginStaff(r.Context(), req.Username, req.Password)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, loginResponse{
		AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: pair.ExpiresIn,
		User: toUserDTO(identity),
	}, "Login berhasil")
}

func (h *Handler) platformLogin(w http.ResponseWriter, r *http.Request) {
	var req adminLoginRequest // {email, password} — bentuk sama, actor beda
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	if req.Email == "" || req.Password == "" {
		httpx.Error(w, httpx.Validation("Email dan password wajib diisi."))
		return
	}
	pair, identity, err := h.svc.LoginPlatform(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, loginResponse{
		AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: pair.ExpiresIn,
		User: toUserDTO(identity),
	}, "Login berhasil")
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	if req.RefreshToken == "" {
		httpx.Error(w, httpx.Validation("refreshToken wajib diisi."))
		return
	}
	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, tokenResponse{
		AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: pair.ExpiresIn,
	}, "Token diperbarui")
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	if req.RefreshToken != "" {
		_ = h.svc.Logout(r.Context(), req.RefreshToken)
	}
	httpx.NoContent(w)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	identity, err := h.svc.Me(r.Context(), authcontract.MustPrincipal(r.Context()))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toUserDTO(identity))
}
