// Package webui menyajikan SPA statis (hasil `vite build` apps/web) yang di-embed ke
// dalam binary Go — sehingga 1 binary = 1 container menyajikan web + API.
//
// Isi dist/:
//   - placeholder.html : di-commit; agar `go build` jalan walau web belum di-build.
//   - index.html, assets/* : hasil build web (disalin saat build produksi; di-gitignore).
//
// Handler menyajikan aset statis apa adanya; untuk path lain (rute klien SPA) ia
// mengembalikan index.html (fallback) agar deep-link & refresh tetap bekerja.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler mengembalikan http.Handler untuk SPA (aset statis + fallback index.html).
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("webui: gagal membuka sub-FS dist: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	// Pilih halaman fallback: index.html (SPA nyata) bila ada, kalau tidak placeholder.html.
	fallback := "index.html"
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		fallback = "placeholder.html"
	}
	fallbackHTML, _ := fs.ReadFile(sub, fallback)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		// Sajikan file statis bila benar-benar ada (mis. /assets/app.js).
		if clean != "." && clean != "" {
			if f, err := sub.Open(clean); err == nil {
				info, statErr := f.Stat()
				_ = f.Close()
				if statErr == nil && !info.IsDir() {
					fileServer.ServeHTTP(w, r)
					return
				}
			}
		}
		// Selain itu = rute klien → kembalikan dokumen SPA.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fallbackHTML)
	})
}
