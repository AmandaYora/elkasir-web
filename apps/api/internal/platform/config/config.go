// Package config memuat konfigurasi 12-factor dari environment + validasi saat boot.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env                string
	Addr               string
	CORSAllowedOrigins []string
	PublicBaseURL      string
	DB                 DB
	JWT                JWT
	Xendit             Xendit
	Storage            ObjectStorage
}

type DB struct {
	DSN string
}

type JWT struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type Xendit struct {
	SecretKey    string
	WebhookToken string
	BaseURL      string
}

// Enabled menandakan jalur QRIS Xendit aktif (secret key terisi).
func (x Xendit) Enabled() bool { return strings.TrimSpace(x.SecretKey) != "" }

// ObjectStorage adalah konfigurasi penyimpanan objek S3-compatible (idcloudhost).
// Objek di-upload public-read, disajikan via URL langsung (lihat PublicURL).
type ObjectStorage struct {
	Endpoint      string // mis. is3.cloudhost.id (tanpa skema)
	Region        string // opsional; default "us-east-1" bila kosong
	Bucket        string // mis. elcodelabs
	AccessKey     string
	SecretKey     string
	UseSSL        bool   // true → https
	BasePath      string // prefix di dalam bucket, mis. elkasir/upload
	PublicBaseURL string // opsional override URL publik; default https://<endpoint>/<bucket>
}

// Enabled menandakan storage aktif (kredensial & target lengkap).
func (o ObjectStorage) Enabled() bool {
	return o.Endpoint != "" && o.Bucket != "" && o.AccessKey != "" && o.SecretKey != ""
}

func (c Config) IsProduction() bool { return c.Env == "production" }

// Load membaca .env (bila ada) lalu environment, memvalidasi nilai wajib.
func Load() (Config, error) {
	_ = godotenv.Load() // .env opsional (di Docker pakai env langsung)

	cfg := Config{
		Env:                getEnv("API_ENV", "development"),
		Addr:               getEnv("API_ADDR", ":8081"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:8080")),
		PublicBaseURL:      getEnv("PUBLIC_BASE_URL", "http://localhost:8081"),
		DB: DB{
			DSN: buildDSN(),
		},
		JWT: JWT{
			Secret:     os.Getenv("JWT_SECRET"),
			AccessTTL:  getDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: getDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Xendit: Xendit{
			SecretKey:    os.Getenv("XENDIT_SECRET_KEY"),
			WebhookToken: os.Getenv("XENDIT_WEBHOOK_TOKEN"),
			BaseURL:      getEnv("XENDIT_BASE_URL", "https://api.xendit.co"),
		},
		Storage: ObjectStorage{
			Endpoint:      getEnv("OBJSTORE_ENDPOINT", ""),
			Region:        getEnv("OBJSTORE_REGION", ""),
			Bucket:        getEnv("OBJSTORE_BUCKET", ""),
			AccessKey:     os.Getenv("OBJSTORE_ACCESS_KEY"),
			SecretKey:     os.Getenv("OBJSTORE_SECRET_KEY"),
			UseSSL:        getBool("OBJSTORE_USE_SSL", true),
			BasePath:      getEnv("OBJSTORE_BASE_PATH", "elkasir/upload"),
			PublicBaseURL: getEnv("OBJSTORE_PUBLIC_BASE_URL", ""),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if strings.TrimSpace(c.DB.DSN) == "" {
		return fmt.Errorf("config: DB_USERNAME & DB_NAME wajib diisi (atau set DB_DSN secara eksplisit)")
	}
	if len(strings.TrimSpace(c.JWT.Secret)) < 16 {
		return fmt.Errorf("config: JWT_SECRET wajib diisi (min 16 karakter)")
	}
	return nil
}

// buildDSN menyusun DSN MySQL dari variabel terpisah (DB_HOST/PORT/USERNAME/PASSWORD/NAME).
// Bila DB_DSN di-set eksplisit, ia dipakai apa adanya (override — mis. untuk PaaS).
// Catatan: password ber-karakter khusus (mis. @ / :) sebaiknya pakai DB_DSN override.
func buildDSN() string {
	if dsn := strings.TrimSpace(os.Getenv("DB_DSN")); dsn != "" {
		return dsn
	}
	user := strings.TrimSpace(os.Getenv("DB_USERNAME"))
	name := strings.TrimSpace(os.Getenv("DB_NAME"))
	if user == "" || name == "" {
		return "" // dilaporkan oleh validate()
	}
	host := getEnv("DB_HOST", "127.0.0.1")
	port := getEnv("DB_PORT", "3306")
	pass := os.Getenv("DB_PASSWORD") // boleh kosong (mis. root Laragon)
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC&charset=utf8mb4&multiStatements=true",
		user, pass, host, port, name)
}

func getEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getBool(key string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
