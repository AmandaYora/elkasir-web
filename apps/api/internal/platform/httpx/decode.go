package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// DecodeJSON membaca body JSON ke dst (menolak field tak dikenal & body berlebih).
// Mengembalikan *APIError siap-tulis bila input tidak valid.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			return BadRequest("Body permintaan terlalu besar.")
		case errors.Is(err, io.EOF):
			return BadRequest("Body permintaan kosong.")
		default:
			return BadRequest("Format JSON tidak valid: " + err.Error())
		}
	}
	if dec.More() {
		return BadRequest("Body hanya boleh berisi satu objek JSON.")
	}
	return nil
}

// QueryInt membaca parameter query integer dengan nilai default.
func QueryInt(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// QueryStr membaca parameter query string dengan nilai default.
func QueryStr(r *http.Request, key, def string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return def
}
