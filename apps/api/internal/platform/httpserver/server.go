package httpserver

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/elkasir/api/internal/platform/config"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter membangun chi router dengan middleware standar (request-id, logging,
// recover→500 JSON, timeout, CORS).
func NewRouter(cfg config.Config) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger)
	r.Use(recoverer)
	r.Use(securityHeaders)
	// Timeout request 30 dtk untuk endpoint biasa; SSE (text/event-stream) dikecualikan
	// agar koneksi push berumur panjang tidak diputus. WriteTimeout server juga dinolkan
	// per-koneksi di handler SSE.
	r.Use(timeoutExceptStream(30 * time.Second))
	r.Use(corsMiddleware(cfg.CORSAllowedOrigins))
	return r
}

// RegisterHealth memasang liveness & readiness probe.
func RegisterHealth(r chi.Router, pool *sql.DB) {
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := pool.PingContext(ctx); err != nil {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "db_unavailable"})
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
}

// NewHTTPServer membungkus handler dengan timeout server yang aman.
func NewHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}
