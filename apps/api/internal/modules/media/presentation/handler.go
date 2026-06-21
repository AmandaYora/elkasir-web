// Package presentation holds the media module's HTTP handler (multipart upload).
package presentation

import (
	"io"
	"net/http"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/media/application"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
)

// maxUpload sedikit di atas batas service (memberi ruang header/overhead multipart).
const maxUpload = 12 << 20

type Handler struct {
	svc  *application.Service
	auth authcontract.Authenticator
}

func NewHandler(svc *application.Service, auth authcontract.Authenticator) *Handler {
	return &Handler{svc: svc, auth: auth}
}

// Routes mounts /uploads (admin-only; owner/admin — selaras dengan write katalog).
func (h *Handler) Routes(r chi.Router) {
	r.Route("/uploads", func(r chi.Router) {
		r.Use(h.auth.Authenticate)
		r.Use(authcontract.RequireActor(authcontract.ActorAdmin))
		r.Use(authcontract.RequireRole("owner", "admin"))
		r.Post("/", h.upload)
	})
}

// upload menerima multipart (field "file" + opsional "category") dan mengembalikan
// { key, url } objek yang tersimpan.
func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		httpx.Error(w, httpx.Validation("Form tidak valid atau file terlalu besar (maks 10 MB)."))
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, httpx.Validation("Field 'file' wajib diisi."))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		httpx.Error(w, httpx.Validation("Gagal membaca file."))
		return
	}

	res, err := h.svc.Upload(r.Context(), r.FormValue("category"), data)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, res)
}
