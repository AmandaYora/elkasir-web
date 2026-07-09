// Package application holds the media module's use cases: stage-2 image compression
// and upload to object storage. Pure-Go pipeline (CGO_ENABLED=0 safe).
package application

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"net/http"
	"strings"

	_ "image/gif"  // register GIF decoder
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	_ "golang.org/x/image/webp" // register WebP decoder (FE may emit WebP)

	"github.com/disintegration/imaging"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
	"github.com/elkasir/api/internal/platform/storage"
)

const (
	maxInputBytes = 10 << 20 // 10 MB — gambar dari FE sudah dikompres tahap-1

	productMaxDim      = 1280 // sisi terpanjang maksimum utk foto produk (tampil di UI web)
	productJPEGQuality = 82   // titik manis ukuran/kualitas untuk foto produk

	// Logo toko dicetak di kertas thermal 384px (58mm) / 576px (80mm) — dan didither
	// hitam-putih di printer, jadi resolusi tinggi tidak menambah kualitas cetak, hanya
	// memperbesar payload yang harus diproses & dikirim ke printer setiap struk. 576
	// mencakup lebar terbesar (80mm) tanpa upscale; mobile men-downscale lagi ke 384
	// bila kertas 58mm. Kualitas dinaikkan sedikit karena ukurannya sudah kecil.
	logoMaxDim      = 576
	logoJPEGQuality = 90
)

// Result adalah balasan upload: key objek + URL publiknya.
type Result struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

type Service struct{ store *storage.Client }

// NewService — store boleh nil (upload nonaktif bila OBJSTORE_* belum diisi).
func NewService(store *storage.Client) *Service { return &Service{store: store} }

// Upload memvalidasi, mengompres tahap-2, lalu menyimpan gambar ke object storage.
func (s *Service) Upload(ctx context.Context, category string, data []byte) (Result, error) {
	if s.store == nil {
		return Result{}, httpx.Internal("Penyimpanan objek belum dikonfigurasi (set OBJSTORE_* di .env).")
	}
	if len(data) == 0 {
		return Result{}, httpx.Validation("File kosong.")
	}
	if len(data) > maxInputBytes {
		return Result{}, httpx.Validation("Ukuran file maksimal 10 MB.")
	}
	if !strings.HasPrefix(http.DetectContentType(data), "image/") {
		return Result{}, httpx.Validation("File harus berupa gambar.")
	}

	maxDim, quality := profileFor(category)
	out, err := compress(data, maxDim, quality)
	if err != nil {
		return Result{}, err
	}

	name := id.New() + ".jpg"
	key, url, err := s.store.Put(ctx, sanitizeCategory(category), name, "image/jpeg", out)
	if err != nil {
		return Result{}, httpx.Internal("Gagal mengunggah gambar ke penyimpanan.")
	}
	return Result{Key: key, URL: url}, nil
}

// profileFor memilih profil kompresi sesuai tujuan pakai gambar: logo toko dicetak di
// struk (kecil, monokrom) butuh resolusi jauh lebih rendah daripada foto produk (tampil
// di UI web, boleh besar & berwarna penuh).
func profileFor(category string) (maxDim, quality int) {
	if sanitizeCategory(category) == "store-logo" {
		return logoMaxDim, logoJPEGQuality
	}
	return productMaxDim, productJPEGQuality
}

// compress: auto-orient (EXIF) → downscale ke maxDim → flatten transparansi ke putih
// → re-encode JPEG pada quality yang diberikan.
func compress(data []byte, maxDim, quality int) ([]byte, error) {
	src, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, httpx.Validation("File bukan gambar yang valid atau formatnya tidak didukung.")
	}
	if b := src.Bounds(); b.Dx() > maxDim || b.Dy() > maxDim {
		src = imaging.Fit(src, maxDim, maxDim, imaging.Lanczos)
	}

	// Flatten alpha (PNG/WebP transparan) ke latar putih agar JPEG tidak menghitam.
	flat := imaging.New(src.Bounds().Dx(), src.Bounds().Dy(), color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	flat = imaging.Overlay(flat, src, image.Pt(0, 0), 1.0)

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, flat, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
		return nil, httpx.Internal("Gagal memproses gambar.")
	}
	return buf.Bytes(), nil
}

// sanitizeCategory membatasi segmen path ke [a-z0-9_-] (cegah path traversal).
func sanitizeCategory(c string) string {
	c = strings.ToLower(strings.TrimSpace(c))
	var b strings.Builder
	for _, r := range c {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 32 {
		out = out[:32]
	}
	if out == "" {
		out = "product"
	}
	return out
}
