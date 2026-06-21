# Rule: API Standard (applies to HTTP boundary — `apps/api/internal/**/presentation`, `apps/web/src/**/services`)

- All business endpoints are versioned under **`/api/v1`**. Health/liveness (`/healthz`, `/readyz`)
  stay at root. The SPA is served from `/` (catch-all, registered last).
- Standard response envelopes (produced centrally by `internal/platform/httpx`):

  ```json
  // success
  { "success": true, "message": "Product created successfully", "data": {} }
  // paginated
  { "success": true, "message": "Data retrieved successfully", "data": [],
    "meta": { "page": 1, "limit": 20, "total": 100, "total_pages": 5 } }
  // error
  { "success": false, "message": "Validation failed", "errors": [] }
  ```

- Handlers must use the `httpx` helpers (`OK`, `Created`, `List`, `NoContent`, `Error`) — do not encode
  ad-hoc JSON shapes. New typed errors go in `httpx` and map to a status + message.
- Frontend consumes these envelopes through the shared Axios client; response interceptors unwrap
  `data` and surface `message`/`errors`.
- Auth: JWT access + refresh. Two actor types — admin web users (roles owner/admin/manager/viewer) and
  POS staff (roles cashier/supervisor). Protected routes require the auth middleware; the principal
  carries `storeId`, `actor`, `role`.
- Pagination query params: `page` (or `offset`) + `limit`; the server clamps `limit` to a safe max.
- Frontend env: `VITE_API_BASE_URL` (dev `http://localhost:8081/api/v1`, prod `/api/v1`). Never hardcode
  the base URL in modules.
