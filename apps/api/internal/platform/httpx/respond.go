package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// JSON menulis payload sebagai JSON dengan status tertentu.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("httpx: encode response", "error", err)
	}
}

// NoContent menulis 204.
func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// successEnvelope adalah bentuk respons sukses standar: { success, message, data }.
type successEnvelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK menulis 200 dalam envelope sukses standar. Pesan opsional (default "OK").
func OK(w http.ResponseWriter, data any, message ...string) {
	JSON(w, http.StatusOK, successEnvelope{Success: true, Message: msgOr(message, "OK"), Data: data})
}

// Created menulis 201 dalam envelope sukses standar. Pesan opsional.
func Created(w http.ResponseWriter, data any, message ...string) {
	JSON(w, http.StatusCreated, successEnvelope{Success: true, Message: msgOr(message, "Created"), Data: data})
}

func msgOr(msgs []string, def string) string {
	if len(msgs) > 0 && msgs[0] != "" {
		return msgs[0]
	}
	return def
}

// errorEnvelope adalah bentuk respons error standar: { success:false, message, errors }.
type errorEnvelope struct {
	Success bool  `json:"success"`
	Message string `json:"message"`
	Errors  []any `json:"errors"`
}

func toErrorEnvelope(e *APIError) errorEnvelope {
	item := map[string]any{"code": e.Code}
	if e.Details != nil {
		item["details"] = e.Details
	}
	return errorEnvelope{Success: false, Message: e.Message, Errors: []any{item}}
}

// Error menulis error dalam envelope terstandar. Error non-APIError → 500 generik
// (detail internal tidak dibocorkan ke klien, hanya dicatat).
func Error(w http.ResponseWriter, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		JSON(w, apiErr.Status, toErrorEnvelope(apiErr))
		return
	}
	slog.Error("httpx: unhandled error", "error", err)
	JSON(w, http.StatusInternalServerError, toErrorEnvelope(Internal("Terjadi kesalahan pada server.")))
}
