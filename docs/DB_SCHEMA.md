# Elkasir — Database Schema

Column-level schema for every table, derived strictly from the migration SQL in
`apps/api/db/migrations/`. For module ownership, conventions, and the primitive-ID rule see
[`knowledge/DATABASE_GUIDE.md`](../knowledge/DATABASE_GUIDE.md).

**Conventions:** engine InnoDB / charset `utf8mb4_unicode_ci`; IDs = `CHAR(26)` ULID; money =
`BIGINT` (rupiah); time = `DATETIME` (UTC). `created_at` defaults `CURRENT_TIMESTAMP`; `updated_at`
adds `ON UPDATE CURRENT_TIMESTAMP`.

**Reference legend**

- **Physical FK** — a real DB `FOREIGN KEY` constraint (only `store_id → stores` tenant keys and a
  few intra-module links remain).
- **Primitive ID** — a plain ID column referencing another module's row, with an index for
  performance but **no FK constraint** (cross-module integrity enforced in the logic layer; see
  migration `000005_drop_cross_module_fks`). These are **NOT physical foreign keys**.

---

## `stores` — shared kernel (tenant root)

PK: `id`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK (ULID). |
| `name` | VARCHAR(150) | NOT NULL. |
| `type` | VARCHAR(60) | default `'F&B'`. |
| `address` | VARCHAR(255) | nullable. |
| `phone` | VARCHAR(40) | nullable. |
| `logo_url` | VARCHAR(500) | nullable — diunggah lewat `POST /uploads?category=store-logo` (migration 000014). |
| `timezone` | VARCHAR(64) | default `'Asia/Jakarta'`. |
| `currency` | CHAR(3) | default `'IDR'`. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

References: none (root). Every other tenant table references this via a physical `store_id` FK.
Profil (`name`/`address`/`phone`/`logo_url`) dibaca-tulis oleh modul `settings` (pengecualian
shared-kernel — lihat `knowledge/DATABASE_GUIDE.md`), disatukan dengan tabel `settings` dalam satu
payload admin `GET/PATCH /settings` dan `GET /pos/config`.

---

## `settings` — shared kernel

PK: `id`. Unique: `store_id`. One row per store (control policy + feature flags).

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (ON DELETE CASCADE), unique. |
| `max_discount_percent` | INT | default 10. |
| `max_operational_expense` | BIGINT | default 200000. |
| `cash_variance_tolerance` | BIGINT | default 5000. |
| `feature_self_order` | TINYINT(1) | default 1. |
| `feature_qris` | TINYINT(1) | default 1. |
| `tax_enabled` | TINYINT(1) | default 0 — aktifkan PPN (migration 000009). |
| `tax_percent` | INT | default 11 — PPN %. |
| `service_percent` | INT | default 2 — biaya layanan %. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

Owned by the `settings` module (`settingsclient` for cross-module reads; admin CRUD `GET/PATCH /settings`).

---

## `admin_users` — adminuser / auth

PK: `id`. Unique: `email`. Index: `store_id`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `name` | VARCHAR(150) | NOT NULL. |
| `email` | VARCHAR(190) | NOT NULL, unique. |
| `password_hash` | VARCHAR(100) | bcrypt. |
| `role` | ENUM(`owner`,`admin`,`manager`,`viewer`) | default `viewer`. |
| `status` | ENUM(`active`,`inactive`) | default `active`. |
| `last_active_at` | DATETIME | nullable. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## `staff` — staff / auth

PK: `id`. Unique: `(store_id, username)`. Index: `store_id`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `name` | VARCHAR(150) | NOT NULL. |
| `username` | VARCHAR(100) | NOT NULL (unique per store). |
| `email` | VARCHAR(190) | nullable. |
| `password_hash` | VARCHAR(100) | bcrypt. |
| `role` | ENUM(`cashier`,`supervisor`) | default `cashier`. |
| `status` | ENUM(`active`,`inactive`) | default `active`. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## `refresh_tokens` — auth

PK: `id`. Unique: `token_hash`. Index: `(actor, subject_id)`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `actor` | ENUM(`admin`,`staff`) | which identity context. |
| `subject_id` | CHAR(26) | **primitive ID** → admin_users(id) or staff(id) depending on `actor`. |
| `token_hash` | CHAR(64) | SHA-256 of the opaque refresh token, unique. |
| `expires_at` | DATETIME | NOT NULL. |
| `revoked_at` | DATETIME | nullable. |
| `created_at` | DATETIME | timestamp. |

---

## `idempotency_keys` — auth / transaction (platform)

PK: `id`. Unique: `(store_id, idempotency_key)`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | tenant scope (no FK declared in this table). |
| `idempotency_key` | VARCHAR(255) | client-supplied key. |
| `request_hash` | CHAR(64) | SHA-256 of the request body. |
| `response_status` | INT | nullable (stored replay status). |
| `response_body` | LONGTEXT | nullable (stored replay body). |
| `created_at` | DATETIME | timestamp. |

---

## `webhook_events` — payment

PK: `id`. Unique: `(provider, event_id)` (dedupe).

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `provider` | VARCHAR(40) | e.g. `tripay`. |
| `event_id` | VARCHAR(255) | provider event id. |
| `processed_at` | DATETIME | default now. |
| `created_at` | DATETIME | timestamp. |

---

## `product_categories` — category

PK: `id`. Unique: `(store_id, name)`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `name` | VARCHAR(120) | NOT NULL (unique per store). |
| `sort_order` | INT | default 0. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## `products` — product

PK: `id`. Unique: `(store_id, sku)`. Indexes: `(store_id, status)`, `category_id`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `category_id` | CHAR(26) | nullable. **physical FK → product_categories(id)** (intra-module, ON DELETE SET NULL). |
| `sku` | VARCHAR(60) | nullable (unique per store). |
| `name` | VARCHAR(150) | NOT NULL. |
| `price` | BIGINT | default 0. |
| `cost` | BIGINT | default 0. |
| `stock` | INT | default 0. |
| `status` | ENUM(`active`,`inactive`) | default `active`. |
| `image_url` | VARCHAR(500) | nullable. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## `dining_tables` — table

PK: `id`. Unique: `(store_id, code)`. `code` encodes the self-order QR (`/order/<code>`).

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `code` | VARCHAR(40) | NOT NULL (unique per store). |
| `name` | VARCHAR(60) | NOT NULL. |
| `area` | VARCHAR(60) | default `''`. |
| `seats` | INT | default 0. |
| `status` | ENUM(`active`,`inactive`) | default `active`. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## `shifts` — shift

PK: `id`. Indexes: `(store_id, status)`, `staff_id`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `staff_id` | CHAR(26) | **primitive ID** → staff(id) (FK dropped in 000005). |
| `status` | ENUM(`open`,`closed`) | default `open`. |
| `initial_cash` | BIGINT | default 0. |
| `cash_sales` | BIGINT | default 0. |
| `qris_sales` | BIGINT | default 0. |
| `additional_capital` | BIGINT | default 0. |
| `expenses` | BIGINT | default 0. |
| `withdrawals` | BIGINT | default 0. |
| `adjustments` | BIGINT | default 0. |
| `drawer_open_count` | INT | default 0. |
| `expected_cash` | BIGINT | nullable (set on close). |
| `actual_cash` | BIGINT | nullable (set on close). |
| `variance` | BIGINT | nullable (set on close). |
| `close_approved_by` | CHAR(26) | nullable. **primitive ID** → staff(id) (FK dropped). |
| `opened_at` | DATETIME | default now. |
| `closed_at` | DATETIME | nullable. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## `transactions` — transaction

PK: `id`. Unique: `(store_id, code)`. Indexes: `(store_id, created_at)`, `shift_id`, `source`,
`status`, `cashier_id`, `self_order_id`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `code` | VARCHAR(40) | NOT NULL (unique per store). |
| `shift_id` | CHAR(26) | nullable. **primitive ID** → shifts(id) (FK dropped). |
| `table_id` | CHAR(26) | nullable. **primitive ID** → dining_tables(id) (FK dropped). |
| `self_order_id` | CHAR(26) | nullable. **primitive ID** → self_orders(id) (circular link, FK dropped). |
| `cashier_id` | CHAR(26) | nullable. **primitive ID** → staff(id) (FK dropped). |
| `order_type` | ENUM(`dineIn`,`takeaway`) | default `takeaway`. |
| `source` | ENUM(`cashier`,`self_order`) | default `cashier`. |
| `payment_method` | ENUM(`cash`,`qris`) | default `cash`. |
| `status` | ENUM(`completed`,`voided`,`refunded`) | default `completed`. |
| `subtotal` | BIGINT | default 0. |
| `discount` | BIGINT | default 0. |
| `tax` | BIGINT | default 0 — PPN. |
| `service_charge` | BIGINT | default 0 — biaya layanan 2% (migration 000010). |
| `gateway_fee` | BIGINT | default 0 — biaya gateway QRIS (0 utk kasir). |
| `total` | BIGINT | default 0 — = subtotal−discount+tax+service_charge+gateway_fee. |
| `amount_received` | BIGINT | default 0. |
| `change_amount` | BIGINT | default 0. |
| `discount_approved_by` | CHAR(26) | nullable. **primitive ID** → staff(id) (FK dropped). |
| `customer_note` | VARCHAR(255) | nullable. |
| `created_at` | DATETIME | timestamp. |

---

## `transaction_items` — transaction (price/name snapshot)

PK: `id`. Index: `transaction_id`. Snapshot of sold items (immune to later product edits).

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `transaction_id` | CHAR(26) | **physical FK → transactions(id)** (intra-module, ON DELETE CASCADE). |
| `product_id` | CHAR(26) | nullable. **primitive ID** → products(id) (FK dropped in 000005). |
| `product_name` | VARCHAR(150) | snapshot. |
| `category` | VARCHAR(120) | default `''` (snapshot). |
| `price` | BIGINT | default 0 (snapshot). |
| `quantity` | INT | default 0. |
| `line_total` | BIGINT | default 0. |
| `note` | VARCHAR(255) | nullable. |
| `created_at` | DATETIME | timestamp. |

---

## `cash_movements` — cashmovement

PK: `id`. Indexes: `shift_id`, `(store_id, created_at)`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `shift_id` | CHAR(26) | nullable. **primitive ID** → shifts(id) (FK dropped). |
| `type` | ENUM(`capital`,`expense`,`adjustment`) | NOT NULL. |
| `amount` | BIGINT | default 0. |
| `notes` | VARCHAR(255) | nullable. |
| `created_by` | CHAR(26) | nullable. **primitive ID** → staff(id) (FK dropped). |
| `approved_by` | CHAR(26) | nullable. **primitive ID** → staff(id) (FK dropped). |
| `created_at` | DATETIME | timestamp. |

---

## `withdrawals` — withdrawal

PK: `id`. Index: `(store_id, created_at)`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `amount` | BIGINT | default 0. |
| `bank` | VARCHAR(80) | default `''`. |
| `account` | VARCHAR(60) | default `''`. |
| `holder` | VARCHAR(120) | default `''`. |
| `status` | ENUM(`pending`,`processing`,`success`,`failed`) | default `pending`. |
| `reference` | VARCHAR(100) | nullable. |
| `requested_by` | CHAR(26) | nullable. **primitive ID** → admin_users(id) (FK dropped). |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## `self_orders` — selforder

PK: `id`. Unique: `claim_code`. Indexes: `(store_id, status)`, `payment_status`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `table_id` | CHAR(26) | nullable. **primitive ID** → dining_tables(id) (FK dropped). |
| `status` | ENUM(`placed`,`preparing`,`completed`) | default `placed`. |
| `payment_method` | ENUM(`qris`,`cash`) | NOT NULL. |
| `payment_status` | ENUM(`pending`,`paid`,`expired`,`failed`,`unpaid`) | NOT NULL. |
| `claim_code` | VARCHAR(40) | nullable, unique (cash pickup code). |
| `subtotal` | BIGINT | default 0. |
| `service_charge` | BIGINT | default 0 — biaya layanan 2% (migration 000010). |
| `gateway_fee` | BIGINT | default 0 — biaya gateway QRIS (0 utk cash). |
| `tax` | BIGINT | default 0 — PPN. |
| `total` | BIGINT | default 0 — = subtotal+service_charge+gateway_fee+tax (yang ditagih). |
| `customer_note` | VARCHAR(255) | nullable. |
| `transaction_id` | CHAR(26) | nullable. **primitive ID** → transactions(id) (circular link, FK dropped). |
| `expires_at` | DATETIME | nullable (QRIS TTL). |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## `self_order_items` — selforder (snapshot)

PK: `id`. Index: `self_order_id`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `self_order_id` | CHAR(26) | **physical FK → self_orders(id)** (intra-module, ON DELETE CASCADE). |
| `product_id` | CHAR(26) | nullable. **primitive ID** → products(id) (FK dropped in 000005). |
| `product_name` | VARCHAR(150) | snapshot. |
| `category` | VARCHAR(120) | default `''`. |
| `price` | BIGINT | default 0. |
| `quantity` | INT | default 0. |
| `line_total` | BIGINT | default 0. |
| `note` | VARCHAR(255) | nullable. |
| `created_at` | DATETIME | timestamp. |

---

## `payments` — payment

PK: `id`. Indexes: `self_order_id`, `provider_ref`. Gateway (Tripay/Midtrans) payment history/reconciliation.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `self_order_id` | CHAR(26) | NOT NULL. **primitive ID** → self_orders(id) (FK dropped in 000005). |
| `provider` | ENUM(`xendit`,`midtrans`,`tripay`) | default `tripay` (migration 000008; older values kept for back-compat). |
| `provider_ref` | VARCHAR(190) | nullable (external ref). |
| `method` | ENUM(`qris`) | default `qris`. |
| `amount` | BIGINT | default 0. |
| `status` | ENUM(`pending`,`paid`,`expired`,`failed`) | default `pending`. |
| `raw_payload` | LONGTEXT | nullable (raw provider payload). |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## Cross-module reference summary (primitive IDs — NOT physical FKs)

These columns were physical FKs originally but were dropped in migration
`000005_drop_cross_module_fks` (columns + indexes retained). They are now logical references only:

| From (table.column) | Logically references | Owning module of target |
|---------------------|----------------------|-------------------------|
| `refresh_tokens.subject_id` | `admin_users.id` / `staff.id` | adminuser / staff |
| `shifts.staff_id` | `staff.id` | staff |
| `shifts.close_approved_by` | `staff.id` | staff |
| `transactions.shift_id` | `shifts.id` | shift |
| `transactions.table_id` | `dining_tables.id` | table |
| `transactions.self_order_id` | `self_orders.id` | selforder |
| `transactions.cashier_id` | `staff.id` | staff |
| `transactions.discount_approved_by` | `staff.id` | staff |
| `transaction_items.product_id` | `products.id` | product |
| `cash_movements.shift_id` | `shifts.id` | shift |
| `cash_movements.created_by` | `staff.id` | staff |
| `cash_movements.approved_by` | `staff.id` | staff |
| `withdrawals.requested_by` | `admin_users.id` | adminuser |
| `self_orders.table_id` | `dining_tables.id` | table |
| `self_orders.transaction_id` | `transactions.id` | transaction |
| `self_order_items.product_id` | `products.id` | product |
| `payments.self_order_id` | `self_orders.id` | selforder |

**Retained physical FKs:** every `store_id → stores(id)` (tenant key, CASCADE), plus the three
intra-module links `products.category_id → product_categories(id)` (SET NULL),
`transaction_items.transaction_id → transactions(id)` (CASCADE), and
`self_order_items.self_order_id → self_orders(id)` (CASCADE).
