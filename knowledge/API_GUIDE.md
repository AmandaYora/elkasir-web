# Elkasir — API Guide

Project-specific HTTP API conventions for the Elkasir backend (`apps/api`, Go modular monolith).
This document describes the **target standard** every endpoint conforms to. For the exact
endpoint catalogue see [`docs/API_CONTRACT.md`](../docs/API_CONTRACT.md); for the data model see
[`knowledge/DATABASE_GUIDE.md`](./DATABASE_GUIDE.md).

---

## 1. Versioning & base path

- **All business endpoints are versioned under `/api/v1`.** Example: `/api/v1/products`,
  `/api/v1/auth/admin/login`, `/api/v1/self-orders`.
- **Infra/probe endpoints stay at the root** (unversioned), so orchestrators and load balancers
  have stable paths:
  - `GET /healthz` — liveness. Returns `{ "status": "ok" }`.
  - `GET /readyz` — readiness; pings the DB. Returns `{ "status": "ready" }` (200) or
    `{ "status": "db_unavailable" }` (503).
- The frontend SPA is served from the root catch-all (`/*`); the REST API lives entirely under
  `/api/v1` so the web routes and API routes never collide in namespace.

---

## 2. Response envelopes

Every JSON response uses one of three envelopes. Money fields are **integer rupiah** (no decimals);
timestamps are **RFC3339 UTC**; IDs are **ULID** strings (26 chars).

### 2.1 Success (single object)

```json
{
  "success": true,
  "message": "Product retrieved.",
  "data": {
    "id": "01HZX2K8Q9...",
    "name": "Es Teh Manis",
    "price": 5000
  }
}
```

### 2.2 Paginated (collection)

```json
{
  "success": true,
  "message": "Products listed.",
  "data": [
    { "id": "01HZX...", "name": "Es Teh Manis", "price": 5000 }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

`data` is **never `null`** for list endpoints — an empty result is `[]`.

### 2.3 Error

```json
{
  "success": false,
  "message": "Stok tidak cukup (Es Teh Manis).",
  "errors": [
    { "code": "unprocessable", "field": "items[0].quantity", "message": "..." }
  ]
}
```

`errors` is an array of structured error detail objects; for simple errors it may carry a single
entry, and `message` always holds a human-readable summary.

---

## 3. Authentication model

Elkasir uses **JWT access + opaque refresh tokens**, with **two distinct actor types** that must
never be mixed.

### 3.1 Actor types

| Actor   | Who                              | Login channel                  | Login field |
|---------|----------------------------------|--------------------------------|-------------|
| `admin` | Dashboard / web back-office users | `POST /api/v1/auth/admin/login` | `email`     |
| `staff` | POS cashiers (Flutter app)        | `POST /api/v1/auth/staff/login` | `username`  |

### 3.2 Roles

| Actor   | Roles                                 | Notes |
|---------|---------------------------------------|-------|
| `admin` | `owner`, `admin`, `manager`, `viewer` | `viewer`/`manager` are read-only on master data; writes require `owner`/`admin`. `owner` only for withdrawals. |
| `staff` | `cashier`, `supervisor`               | `supervisor` typically approves over-tolerance closes / over-policy discounts. |

### 3.3 Tokens

- **Access token** — JWT (HS256). Claims: `sub` (subject id), `store_id`, `actor`, `role`, `iat`,
  `exp`. Sent on every request as `Authorization: Bearer <accessToken>`.
- **Refresh token** — opaque random hex string. Only its SHA-256 hash is stored server-side
  (`refresh_tokens` table). Refresh **rotates**: the old token is revoked and a new pair issued.
- Login response payload:

  ```json
  {
    "accessToken": "<jwt>",
    "refreshToken": "<opaque>",
    "expiresIn": 900,
    "user": { "id": "...", "name": "...", "email": "...", "role": "owner", "storeId": "...", "actor": "admin" }
  }
  ```

  `expiresIn` is the access-token lifetime in seconds. (Refresh returns the same shape minus
  `user`.)

### 3.4 Endpoints

| Endpoint                          | Auth     | Purpose |
|-----------------------------------|----------|---------|
| `POST /api/v1/auth/admin/login`   | public   | Admin (web) login by email+password. |
| `POST /api/v1/auth/staff/login`   | public   | Staff (POS) login by username+password. |
| `POST /api/v1/auth/refresh`       | public   | Rotate tokens using a valid `refreshToken`. |
| `POST /api/v1/auth/logout`        | public   | Revoke the supplied `refreshToken` (idempotent, returns 204). |
| `GET  /api/v1/auth/me`            | Bearer   | Identity of the current principal. |

### 3.5 Guard middleware (server-side enforcement)

- `Authenticate` — validates the Bearer access token; on success injects the `Principal`
  (`{ SubjectID, StoreID, Actor, Role }`) into the request context. Missing/invalid → `401`.
- `RequireActor(actor)` — rejects with `403` if the principal's actor type differs (e.g. master
  data endpoints require `actor=admin`; POS write endpoints require `actor=staff`).
- `RequireRole(roles...)` — rejects with `403` if the principal's role is not in the allowed set.

### 3.6 Multi-tenancy

Every business row is scoped by **`store_id`**. The store is **derived from the authenticated
principal's `store_id` claim — never read from the request body or query string.** Service calls
always pass `MustPrincipal(ctx).StoreID`, guaranteeing tenant isolation.

---

## 4. Pagination

List endpoints accept these query params (read by `PageFromRequest`):

| Param    | Meaning                                                | Default |
|----------|--------------------------------------------------------|---------|
| `limit`  | Page size (clamped to a per-endpoint max).             | 20 (50 for self-orders) |
| `offset` | Row offset.                                            | 0 |
| `page`   | 1-based page number; when `> 1` and `offset` is unset, offset is computed as `(page-1)*limit`. | — |

Per-endpoint maximums: most lists cap `limit` at **100**; self-order incoming list caps at **200**.
Some admin lists (categories, tables, staff, admin-users) return all rows in a single page.

Responses use the **paginated envelope** (`data` + `meta`). `meta` exposes `page`, `limit`,
`total`, and `total_pages`.

---

## 5. Filtering query params (where supported)

- **Products** (`GET /api/v1/products`): `status`, `categoryId`, `search`.
- **Transactions** (`GET /api/v1/transactions`): `status`, `source`, `paymentMethod`, `search`,
  `from`, `to` (RFC3339 or `YYYY-MM-DD`).
- **Reports** (`GET /api/v1/reports/*`): `from`, `to` (default range = last 30 days);
  `top-products` also accepts `limit` (default 10).
- **Self-orders** (`GET /api/v1/self-orders`): `status`.

---

## 6. Idempotency

- `POST /api/v1/transactions` **requires** an `Idempotency-Key` request header. The server hashes
  the raw body; a replay with the same key + same body returns the original transaction (`200`),
  a different body with the same key returns `409 conflict`.
- Self-order cash checkout (`POST /api/v1/self-orders/redeem/{claimCode}/checkout`) is idempotent
  via order state (an already-paid order replays its result); it reads an optional
  `Idempotency-Key` header.

---

## 7. Error codes

Stable, client-consumable error codes (carried in the error envelope) and their HTTP status:

| Code               | HTTP | Meaning |
|--------------------|------|---------|
| `bad_request`      | 400  | Malformed request / missing required header. |
| `validation_error` | 400  | Field validation failed. |
| `unauthorized`     | 401  | Missing/invalid/expired token, invalid session. |
| `forbidden`        | 403  | Wrong actor, insufficient role, or policy approval required. |
| `not_found`        | 404  | Resource not found (tenant-scoped). |
| `conflict`         | 409  | Unique constraint / idempotency / state conflict. |
| `unprocessable`    | 422  | Semantically invalid (e.g. insufficient stock). |
| `rate_limited`     | 429  | Rate limit exceeded (public self-order endpoints). |
| `internal`         | 500  | Unhandled server error (details not leaked to client). |

---

## 8. Public (no-auth) self-order endpoints

Customers scan a QR encoding their table `code` and order without logging in. These endpoints are
**unauthenticated** and **rate-limited per IP** (60 req window):

| Endpoint                                              | Purpose |
|-------------------------------------------------------|---------|
| `GET  /api/v1/public/order/{tableCode}`               | Fetch the menu for a table (table info + categories + active products). |
| `POST /api/v1/public/order/{tableCode}`               | Place a self-order (QRIS → returns QR string; cash → returns a claim code to redeem at the cashier). |
| `GET  /api/v1/public/order/status/{selfOrderId}`      | Poll order + payment status. |
| `POST /api/v1/public/order/{selfOrderId}/simulate-paid` | **DEV only** — mark a pending QRIS order paid without a real gateway (returns `404` when a live payment provider is enabled). |

A separate **webhook** endpoint (`POST /api/v1/webhooks/payment`) is also unauthenticated at the
middleware layer — its authenticity is verified inside the `payment` module by the **active
provider**: Tripay via the `X-Callback-Signature` header (HMAC-SHA256 of the raw body with the
private key), or Midtrans via the `signature_key` field
(`SHA512(order_id + status_code + gross_amount + ServerKey)`).

Admin/staff-facing self-order management endpoints (list incoming, update status, redeem, redeem
checkout) **do** require authentication. See [`docs/API_CONTRACT.md`](../docs/API_CONTRACT.md).
