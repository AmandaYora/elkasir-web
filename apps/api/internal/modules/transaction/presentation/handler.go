// Package presentation holds the transaction module's HTTP handlers and routes.
package presentation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/transaction/application"
	"github.com/elkasir/api/internal/modules/transaction/domain"
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

// Routes: POST /transactions (staf POS, idempoten); GET list/detail (admin & staf).
func (h *Handler) Routes(r chi.Router) {
	r.Route("/transactions", func(r chi.Router) {
		r.Use(h.auth.Authenticate)

		r.Get("/", h.list)
		r.Get("/{id}", h.get)

		r.Group(func(r chi.Router) {
			r.Use(authcontract.RequireActor(authcontract.ActorStaff))
			r.Post("/", h.create)
		})
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		httpx.Error(w, httpx.BadRequest("Header Idempotency-Key wajib untuk pembuatan transaksi."))
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		httpx.Error(w, httpx.BadRequest("Body permintaan terlalu besar atau tidak terbaca."))
		return
	}
	sum := sha256.Sum256(body)
	reqHash := hex.EncodeToString(sum[:])

	var in application.CreateInput
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		httpx.Error(w, httpx.BadRequest("Format JSON tidak valid: "+err.Error()))
		return
	}

	dto, created, err := h.svc.Create(r.Context(), authcontract.MustPrincipal(r.Context()), idemKey, reqHash, in)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if created {
		httpx.Created(w, dto, "Transaksi berhasil dibuat")
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	storeID := authcontract.MustPrincipal(r.Context()).StoreID
	dto, err := h.svc.Get(r.Context(), storeID, chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page := httpx.PageFromRequest(r, 20, 100)
	f := domain.ListFilter{
		StoreID:       authcontract.MustPrincipal(r.Context()).StoreID,
		Status:        httpx.QueryStr(r, "status", ""),
		Source:        httpx.QueryStr(r, "source", ""),
		PaymentMethod: httpx.QueryStr(r, "paymentMethod", ""),
		Search:        httpx.QueryStr(r, "search", ""),
		From:          parseTime(httpx.QueryStr(r, "from", "")),
		To:            parseTime(httpx.QueryStr(r, "to", "")),
		Limit:         page.Limit,
		Offset:        page.Offset,
	}
	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, page))
}

func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
