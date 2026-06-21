package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Root harus mengembalikan dokumen HTML (index.html bila ada, kalau tidak placeholder).
func TestHandler_ServesDocumentAtRoot(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("GET / content-type = %q, want text/html", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Fatal("GET / body kosong")
	}
}

// Rute klien (deep-link) yang bukan file harus fallback ke dokumen SPA (200 HTML),
// bukan 404 — agar refresh/deep-link bekerja.
func TestHandler_SpaFallbackForClientRoute(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/products/some/deep/route")
	if err != nil {
		t.Fatalf("GET deep route: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fallback status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("fallback content-type = %q, want text/html", ct)
	}
}
