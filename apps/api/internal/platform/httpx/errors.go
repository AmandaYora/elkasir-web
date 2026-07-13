// Package httpx menyediakan format error API terstandar + helper response/decode.
package httpx

import "net/http"

// Kode error stabil (dikonsumsi klien). Format envelope:
//
//	{ "error": { "code": "string", "message": "string", "details": {} } }
const (
	CodeBadRequest      = "bad_request"
	CodeValidation      = "validation_error"
	CodeUnauthorized    = "unauthorized"
	CodeForbidden       = "forbidden"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodeUnprocessable   = "unprocessable"
	CodeRateLimited     = "rate_limited"
	CodeInternal        = "internal"
	CodePaymentRequired = "payment_required"
)

// APIError adalah error yang dapat dipetakan langsung ke response HTTP.
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *APIError) Error() string { return e.Message }

// WithDetails mengembalikan salinan error dengan detail tambahan (mis. field errors).
func (e *APIError) WithDetails(details any) *APIError {
	cp := *e
	cp.Details = details
	return &cp
}

func newErr(status int, code, msg string) *APIError {
	return &APIError{Status: status, Code: code, Message: msg}
}

func BadRequest(msg string) *APIError { return newErr(http.StatusBadRequest, CodeBadRequest, msg) }
func Validation(msg string) *APIError { return newErr(http.StatusBadRequest, CodeValidation, msg) }
func Unauthorized(msg string) *APIError {
	return newErr(http.StatusUnauthorized, CodeUnauthorized, msg)
}
func Forbidden(msg string) *APIError { return newErr(http.StatusForbidden, CodeForbidden, msg) }
func NotFound(msg string) *APIError  { return newErr(http.StatusNotFound, CodeNotFound, msg) }
func Conflict(msg string) *APIError  { return newErr(http.StatusConflict, CodeConflict, msg) }
func Unprocessable(msg string) *APIError {
	return newErr(http.StatusUnprocessableEntity, CodeUnprocessable, msg)
}
func RateLimited(msg string) *APIError {
	return newErr(http.StatusTooManyRequests, CodeRateLimited, msg)
}
func Internal(msg string) *APIError { return newErr(http.StatusInternalServerError, CodeInternal, msg) }

// PaymentRequired (402) — deliberately distinct from Forbidden (403), so the frontend can tell
// "unauthorized" apart from "tenant subscription unpaid" and redirect accordingly (PLAN.md §2.15).
func PaymentRequired(msg string) *APIError {
	return newErr(http.StatusPaymentRequired, CodePaymentRequired, msg)
}
