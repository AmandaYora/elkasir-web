# Elkasir — API Contract

Full endpoint catalogue for the Elkasir backend, grouped by module. Conventions
(`/api/v1` versioning, response envelopes, auth, pagination, error codes) are defined in
[`knowledge/API_GUIDE.md`](../knowledge/API_GUIDE.md).

**Conventions recap**

- Base path: `/api/v1` (health probes `GET /healthz`, `GET /readyz` are at root).
- Success (single): `{ "success": true, "message": "...", "data": {…} }`.
- Paginated: `{ "success": true, "message": "...", "data": [...], "meta": { page, limit, total, total_pages } }`.
- Error: `{ "success": false, "message": "...", "errors": [...] }`.
- Auth: `Authorization: Bearer <accessToken>`. `store_id` is taken from the principal, never the body.
- Money = integer rupiah; IDs = ULID; timestamps = RFC3339 UTC.

Auth column legend: **public** (no token), **Bearer** (any authenticated actor),
**admin** (`actor=admin`), **staff** (`actor=staff`), plus role guards where noted.

---

## Auth — `/api/v1/auth`

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| POST | `/api/v1/auth/admin/login` | public | Admin/web login. |
| POST | `/api/v1/auth/staff/login` | public | Staff/POS login. |
| POST | `/api/v1/auth/refresh` | public | Rotate token pair. |
| POST | `/api/v1/auth/logout` | public | Revoke a refresh token. |
| GET  | `/api/v1/auth/me` | Bearer | Current principal identity. |

**`POST /auth/admin/login`** — body `{ "email": string, "password": string }`.
`data` →
```json
{ "accessToken": "<jwt>", "refreshToken": "<opaque>", "expiresIn": 900,
  "user": { "id": "...", "name": "...", "email": "...", "role": "owner", "storeId": "...", "actor": "admin" } }
```

**`POST /auth/staff/login`** — body `{ "username": string, "password": string }`. `data` shape as
above (`user.actor` = `"staff"`, `user.role` = `cashier|supervisor`).

**`POST /auth/refresh`** — body `{ "refreshToken": string }`. `data` →
`{ "accessToken", "refreshToken", "expiresIn" }` (no `user`). Invalid/expired/revoked → `401`.

**`POST /auth/logout`** — body `{ "refreshToken": string }` (optional). Revokes the token
(idempotent). Returns `204` (no body).

**`GET /auth/me`** — no body. `data` → the `user` object (`id, name, email?, role, storeId, actor`).

---

## Products — `/api/v1/products`

Guard: `Authenticate` + `RequireActor(admin)`; writes additionally `RequireRole(owner, admin)`.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/products` | admin | List products (paginated). |
| GET | `/api/v1/products/{id}` | admin | Get one product. |
| POST | `/api/v1/products` | owner/admin | Create product. |
| PUT | `/api/v1/products/{id}` | owner/admin | Update product. |
| DELETE | `/api/v1/products/{id}` | owner/admin | Delete product (`204`). |
| POST | `/api/v1/products/{id}/adjust-stock` | owner/admin | Adjust stock by a delta. |

- **List** query params: `status`, `categoryId`, `search`, `limit` (def 20, max 100), `offset`, `page`.
- **Request body (`ProductInput`, create/update):**
  `{ "categoryId"?: string, "sku": string, "name": string (required), "price": int>=0, "cost": int>=0, "stock": int>=0, "status": "active"|"inactive", "imageUrl"?: string }`.
  Duplicate SKU → `409 conflict`.
- **adjust-stock body:** `{ "delta": int }` (signed).
- **Response `data` (Product):**
  `{ "id", "name", "sku", "categoryId", "category", "price", "cost", "stock", "status", "imageUrl", "createdAt" }`.

---

## Categories — `/api/v1/categories`

Guard: `Authenticate` + `RequireActor(admin)`; writes `RequireRole(owner, admin)`.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/categories` | admin | List categories (with product counts). |
| GET | `/api/v1/categories/{id}` | admin | Get one category. |
| POST | `/api/v1/categories` | owner/admin | Create category. |
| PUT | `/api/v1/categories/{id}` | owner/admin | Update category. |
| DELETE | `/api/v1/categories/{id}` | owner/admin | Delete category (`204`). |

- **Body (`CategoryInput`):** `{ "name": string (required), "sortOrder"?: int }`. Duplicate name → `409`.
- **Response `data` (Category):** `{ "id", "name", "sortOrder", "productCount", "createdAt" }`.

---

## Tables (dining tables) — `/api/v1/tables`

Guard: `Authenticate` + `RequireActor(admin)`; writes `RequireRole(owner, admin)`.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/tables` | admin | List tables. |
| GET | `/api/v1/tables/{id}` | admin | Get one table. |
| POST | `/api/v1/tables` | owner/admin | Create table. |
| PUT | `/api/v1/tables/{id}` | owner/admin | Update table. |
| DELETE | `/api/v1/tables/{id}` | owner/admin | Delete table (`204`). |

- **Body (`TableInput`):** `{ "code": string (required), "name": string, "area": string, "seats": int>=0, "status": "active"|"inactive" }`. `name` defaults to `code` if blank. Duplicate code → `409`.
- **Response `data` (DiningTable):** `{ "id", "code", "name", "area", "seats", "status", "createdAt" }`.
- `code` is what the self-order QR encodes (`/order/<code>`).

---

## Staff (POS users) — `/api/v1/staff`

Guard: `Authenticate` + `RequireActor(admin)`; writes `RequireRole(owner, admin)`.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/staff` | admin | List staff. |
| GET | `/api/v1/staff/{id}` | admin | Get one staff. |
| POST | `/api/v1/staff` | owner/admin | Create staff. |
| PUT | `/api/v1/staff/{id}` | owner/admin | Update staff (no password). |
| POST | `/api/v1/staff/{id}/reset-password` | owner/admin | Reset password (`204`). |
| DELETE | `/api/v1/staff/{id}` | owner/admin | Delete staff (`204`). |

- **Create body:** `{ "name", "username", "email"?, "password" (min 6), "role": "cashier"|"supervisor", "status": "active"|"inactive" }`. Duplicate username → `409`.
- **Update body:** same minus `password`.
- **reset-password body:** `{ "password": string (min 6) }`.
- **Response `data` (Staff):** `{ "id", "name", "username", "email", "role", "status", "createdAt" }` (password hash never exposed).

---

## Admin users (web) — `/api/v1/admin-users`

Guard: `Authenticate` + `RequireActor(admin)`; writes `RequireRole(owner, admin)`.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/admin-users` | admin | List admin users. |
| GET | `/api/v1/admin-users/{id}` | admin | Get one admin user. |
| POST | `/api/v1/admin-users` | owner/admin | Create admin user. |
| PUT | `/api/v1/admin-users/{id}` | owner/admin | Update admin user (no password). |
| POST | `/api/v1/admin-users/{id}/reset-password` | owner/admin | Reset password (`204`). |
| DELETE | `/api/v1/admin-users/{id}` | owner/admin | Delete admin user (`204`). |

- **Create body:** `{ "name", "email" (required), "password" (min 6), "role": "owner"|"admin"|"manager"|"viewer", "status": "active"|"inactive" }`. Duplicate email → `409`.
- **Update body:** `{ "name", "email", "role", "status" }`.
- **reset-password body:** `{ "password": string (min 6) }`.
- **Response `data` (AdminUser):** `{ "id", "name", "email", "role", "status", "lastActiveAt"?, "createdAt" }`.

---

## Transactions — `/api/v1/transactions`

Guard: `Authenticate` (read by any actor); **create requires `RequireActor(staff)`**.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/transactions` | Bearer | List transactions (paginated, filterable). |
| GET | `/api/v1/transactions/{id}` | Bearer | Get one transaction (with items). |
| POST | `/api/v1/transactions` | staff | Create a sale (atomic: decrements stock + records sale). |

- **List query params:** `status`, `source`, `paymentMethod`, `search`, `from`, `to` (RFC3339 or `YYYY-MM-DD`), `limit` (def 20, max 100), `offset`, `page`.
- **Create** requires header **`Idempotency-Key`** (missing → `400`). Body max 1 MiB; unknown JSON fields → `400`.
  Body (`CreateInput`):
  ```json
  {
    "items": [ { "productId": string, "quantity": int>0, "note"?: string } ],
    "discount": int>=0,
    "paymentMethod": "cash"|"qris",
    "amountReceived": int,
    "tableId"?: string,
    "orderType": "dineIn"|"takeaway",
    "discountApprovedBy"?: string,
    "customerNote"?: string
  }
  ```
  Rules: cashier id = the staff principal; cash `amountReceived` must cover the total (else `400`);
  discount over the store policy without `discountApprovedBy` → `403`; insufficient stock → `422`;
  idempotency-key reuse with a different body → `409`.
- **Status:** `201` on a newly created transaction, `200` on an idempotent replay.
- **Response `data` (Transaction):**
  `{ "id", "code", "shiftId"?, "tableId"?, "selfOrderId"?, "cashierId"?, "orderType", "source", "paymentMethod", "status", "subtotal", "discount", "tax", "total", "amountReceived", "changeAmount", "customerNote"?, "createdAt", "items": [ { "productId"?, "productName", "category", "price", "quantity", "lineTotal", "note"? } ] }`.
  (List rows omit item detail.)

---

## Shifts — `/api/v1/shifts`

Guard: `Authenticate` (read by any actor); **open/close require `RequireActor(staff)`**.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/shifts` | Bearer | List shifts (paginated). |
| GET | `/api/v1/shifts/current` | Bearer | Current open shift (`204` if none). |
| GET | `/api/v1/shifts/{id}` | Bearer | Get one shift. |
| POST | `/api/v1/shifts` | staff | Open a shift (`201`). |
| POST | `/api/v1/shifts/{id}/close` | staff | Close a shift. |

- **Open body:** `{ "initialCash": int }`. Fails `409` if a shift is already open.
- **Close body:** `{ "actualCash": int, "drawerOpenCount": int, "closeApprovedBy"?: string }`. Computes
  expected cash + variance; variance over tolerance without `closeApprovedBy` → `403`; already
  closed → `409`.
- **Response `data` (Shift):**
  `{ "id", "staffId", "status": "open"|"closed", "initialCash", "cashSales", "qrisSales", "additionalCapital", "expenses", "withdrawals", "adjustments", "drawerOpenCount", "expectedCash"?, "actualCash"?, "variance"?, "closeApprovedBy"?, "openedAt", "closedAt"?, "createdAt" }`.

---

## Cash movements — `/api/v1/cash-movements`

Guard: `Authenticate` (any authenticated actor).

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/cash-movements` | Bearer | List cash movements (paginated). |
| POST | `/api/v1/cash-movements` | Bearer | Record capital/expense/adjustment (`201`). |

- **Body (`CashMovementInput`):** `{ "type": "capital"|"expense"|"adjustment", "amount": int, "notes"?: string, "approvedBy"?: string }`.
  Capital/expense must be `> 0`; adjustment must be non-zero (may be negative). Expense over the
  store expense plafond without `approvedBy` → `403`. Attributed to the current open shift.
- **Response `data` (CashMovement):** `{ "id", "shiftId"?, "type", "amount", "notes"?, "createdBy"?, "approvedBy"?, "createdAt" }`.

---

## Withdrawals — `/api/v1/withdrawals`

Guard: `Authenticate` + `RequireActor(admin)`; **create additionally `RequireRole(owner)`**.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/withdrawals` | admin | List withdrawals (paginated). |
| POST | `/api/v1/withdrawals` | owner | Request a payout (`201`). |

- **Body (`WithdrawalInput`):** `{ "amount": int>0, "bank": string, "account": string, "holder": string }` (all required).
- **Response `data` (Withdrawal):** `{ "id", "amount", "bank", "account", "holder", "status": "pending"|"processing"|"success"|"failed", "reference"?, "requestedBy"?, "createdAt" }`.

---

## Reports — `/api/v1/reports`

Guard: `Authenticate` (any authenticated actor). All read-only `GET`. Common query params:
`from`, `to` (RFC3339 or `YYYY-MM-DD`; default range = last 30 days).

| Method | Path | Auth | Extra params | `data` |
|--------|------|------|--------------|--------|
| GET | `/api/v1/reports/dashboard` | Bearer | `from`, `to` | `{ "summary": { txCount, revenue, cashTotal, qrisTotal }, "recent": [ { id, code, source, paymentMethod, total, createdAt } ] }` |
| GET | `/api/v1/reports/sales` | Bearer | `from`, `to` | `[ { "day", "txCount", "revenue" } ]` |
| GET | `/api/v1/reports/top-products` | Bearer | `from`, `to`, `limit` (def 10) | `[ { "productName", "qty", "revenue" } ]` |
| GET | `/api/v1/reports/sales-by-category` | Bearer | `from`, `to` | `[ { "category", "revenue", "qty" } ]` |
| GET | `/api/v1/reports/payment-distribution` | Bearer | `from`, `to` | `{ "cashTotal", "qrisTotal", "cashCount", "qrisCount" }` |
| GET | `/api/v1/reports/staff-performance` | Bearer | `from`, `to` | `[ { "staffId", "name", "txCount", "revenue" } ]` |

All figures count only `status = completed` transactions.

---

## Settings — `/api/v1/settings` + `/api/v1/pos`

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/settings` | admin | Store profile + control policy + feature flags. |
| PATCH | `/api/v1/settings` | admin (owner/admin) | Update the same (full object — client sends the whole form). |
| GET | `/api/v1/pos/pricing` | Bearer | Read-only subset (service % + PPN) for POS math. |
| GET | `/api/v1/pos/config` | Bearer | Full POS bootstrap payload: store identity + pricing + feature flags + approval thresholds. Pulled by the mobile app on login/refresh. |

- **Settings `data`:** `{ "storeName", "storePhone", "storeAddress", "storeLogoUrl", "maxDiscountPercent", "maxOperationalExpense", "cashVarianceTolerance", "featureSelfOrder", "featureQris", "featurePayAtCashier", "taxEnabled", "taxPercent", "servicePercent" }`. `storeLogoUrl` is set by first uploading via `POST /uploads` (`category=store-logo`) and PATCHing the returned `url` here. `storeName` required (non-empty); `PATCH` validates percent fields 0–100 and rejects self-order active with no payment method enabled.
- **`/pos/config` `data`:** `{ "store": { "name", "phone", "address", "logoUrl" }, "pricing": { "servicePercent", "taxPercent", "taxEnabled" }, "features": { "qris", "selfOrder", "payAtCashier" }, "thresholds": { "maxDiscountPercent", "maxOperationalExpense", "cashVarianceTolerance" } }`.

---

## Self-orders (admin/staff) — `/api/v1/self-orders`

Guard: `Authenticate` (any authenticated actor).

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/self-orders` | Bearer | List incoming self-orders (paginated, def 50 / max 200). |
| PATCH | `/api/v1/self-orders/{id}/status` | Bearer | Update order status. |
| GET | `/api/v1/self-orders/redeem/{claimCode}` | Bearer | Look up a cash order by claim code. |
| POST | `/api/v1/self-orders/redeem/{claimCode}/checkout` | Bearer | Settle a cash self-order at the cashier (atomic fulfilment). |

- **List** query param: `status` (optional filter: `placed|preparing|completed`).
- **Update-status body:** `{ "status": "placed"|"preparing"|"completed" }`. Unknown status → `400`; not found → `404`.
- **Redeem** path param `{claimCode}`; not found → `404`. `data` → SelfOrder.
- **Redeem checkout** reads an optional `Idempotency-Key` header; idempotent via order state (an
  already-paid order replays). Non-cash order → `400`. Atomically decrements stock + records a
  `self_order` sale + marks the order paid/completed. `data` → `{ "transactionId", "order": SelfOrder }`.
- **SelfOrder `data`:** `{ "id", "tableCode", "tableName", "status", "paymentMethod", "paymentStatus", "claimCode"?, "subtotal", "total", "customerNote"?, "transactionId"?, "createdAt", "items": [ { "productName", "category", "price", "quantity", "lineTotal", "note"? } ] }`.

---

## Public self-order (no auth) — `/api/v1/public/order`

Unauthenticated, **rate-limited per IP**. Customers reach these by scanning a table QR.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/public/order/{tableCode}` | public | Menu for the table. |
| POST | `/api/v1/public/order/{tableCode}` | public | Place a self-order (`201`). |
| GET | `/api/v1/public/order/status/{selfOrderId}` | public | Poll order/payment status. |
| POST | `/api/v1/public/order/{selfOrderId}/simulate-paid` | public | **DEV** — mark pending QRIS paid (no gateway). |

- **Menu** `data` (PublicMenu): `{ "table": { code, name, area, status }, "categories": [string], "products": [ { id, name, category, price, imageUrl? } ] }`. Unknown table → `404`.
- **Place** body (`PlaceInput`): `{ "items": [ { "productId": string, "quantity": int>0, "note"?: string } ], "paymentMethod": "qris"|"cash", "customerNote"?: string }`.
  - QRIS → order `payment_status = pending`, `data` includes `qrString` (and `simulated` in dev).
  - Cash → order `payment_status = unpaid` with a `claimCode` (redeem at cashier).
  - `data` (PlaceResult): `{ "order": SelfOrder, "qrString"?, "claimCode"?, "simulated"? }`.
  - Inactive/unknown table → `404`/`422`; empty items / bad payment method / qty ≤ 0 / unknown or inactive product → `400`/`422`.
- **Status** `data` (SelfOrderStatus): `{ "id", "status", "paymentStatus", "total" }`. Not found → `404`.
- **simulate-paid** returns `data` → `{ "status": "paid" }`; **`404` when a live payment provider is enabled** (QRIS-only orders may be simulated). Marks the order paid and triggers fulfilment.

### Payment webhook (no auth middleware)

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| POST | `/api/v1/webhooks/payment` | provider signature (verified in payment module) | Receive QRIS payment callbacks (Tripay/Midtrans, provider-agnostic). |

Verifies the provider token, dedupes by event id (`webhook_events`), and on a paid event triggers
self-order fulfilment (stock decrement + transaction). Returns `data` → `{ "received": "ok" }`
(always `200` unless the token is invalid → `401`, or a transient failure → `500`, so the provider
retries).
