package httpx

import "net/http"

// Page adalah parameter paginasi (limit/offset) ternormalisasi.
type Page struct {
	Limit  int
	Offset int
}

// PageFromRequest membaca ?limit & ?offset (atau ?page) dengan batas aman.
func PageFromRequest(r *http.Request, defLimit, maxLimit int) Page {
	limit := QueryInt(r, "limit", defLimit)
	if limit <= 0 {
		limit = defLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	offset := QueryInt(r, "offset", 0)
	if page := QueryInt(r, "page", 0); page > 1 && offset == 0 {
		offset = (page - 1) * limit
	}
	if offset < 0 {
		offset = 0
	}
	return Page{Limit: limit, Offset: offset}
}

// Paginated adalah envelope daftar standar: { success, message, data, meta }.
type Paginated[T any] struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    []T            `json:"data"`
	Meta    PaginationMeta `json:"meta"`
}

// PaginationMeta mengikuti standar API: page / limit / total / total_pages.
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// List membungkus data + metadata paginasi standar (data tak pernah null). Pesan opsional.
func List[T any](data []T, total int64, p Page, message ...string) Paginated[T] {
	if data == nil {
		data = []T{}
	}
	page := 1
	totalPages := 0
	if p.Limit > 0 {
		page = p.Offset/p.Limit + 1
		totalPages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}
	return Paginated[T]{
		Success: true,
		Message: msgOr(message, "Data retrieved successfully"),
		Data:    data,
		Meta:    PaginationMeta{Page: page, Limit: p.Limit, Total: total, TotalPages: totalPages},
	}
}
