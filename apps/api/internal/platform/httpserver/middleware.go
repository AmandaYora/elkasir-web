package httpserver

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// securityHeaders menambahkan header keamanan dasar pada setiap response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// RateLimit adalah pembatas laju per-IP (fixed window) sederhana untuk endpoint
// publik — mencegah penyalahgunaan tanpa dependensi eksternal.
func RateLimit(maxPerMinute int) func(http.Handler) http.Handler {
	type bucket struct {
		count int
		reset time.Time
	}
	var (
		mu      sync.Mutex
		clients = map[string]*bucket{}
	)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			now := time.Now()
			mu.Lock()
			b := clients[ip]
			if b == nil || now.After(b.reset) {
				b = &bucket{count: 0, reset: now.Add(time.Minute)}
				clients[ip] = b
			}
			b.count++
			over := b.count > maxPerMinute
			mu.Unlock()
			if over {
				httpx.Error(w, httpx.RateLimited("Terlalu banyak permintaan. Coba lagi sebentar."))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requestLogger mencatat tiap request secara terstruktur (slog).
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

// recoverer mengubah panic di handler menjadi 500 JSON terstandar (bukan crash).
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil && rec != http.ErrAbortHandler {
				slog.Error("panic_recovered",
					"panic", rec,
					"path", r.URL.Path,
					"request_id", middleware.GetReqID(r.Context()),
					"stack", string(debug.Stack()),
				)
				httpx.Error(w, httpx.Internal("Terjadi kesalahan pada server."))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware mengizinkan origin web dev (mis. :8080) memanggil API.
func corsMiddleware(origins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key", "X-Requested-With"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: false,
		MaxAge:           300,
	})
}
