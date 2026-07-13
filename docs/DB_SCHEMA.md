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
| `slug` | VARCHAR(60) | NOT NULL, unique (migration 000016). Identitas tenant di URL self-order publik (`/order/<slug>/<kodeMeja>`) — fix untuk kode meja yang cuma unik per-toko. Ditulis modul `platform` saat tenant dibuat; dibaca (read-only) modul `settings` untuk ditampilkan admin. |
| `status` | ENUM(`active`,`suspended`) | default `active` (migration 000016). Ditulis modul `platform`; dibaca `auth` di setiap request+login/refresh untuk menegakkan blokir akses tenant yang di-suspend (§2.13, `auth/infrastructure/middleware.go` — read langsung, tanpa cache). |
| `type` | VARCHAR(60) | default `'F&B'`. |
| `address` | VARCHAR(255) | nullable. |
| `phone` | VARCHAR(40) | nullable. |
| `logo_url` | VARCHAR(500) | nullable — diunggah lewat `POST /uploads?category=store-logo` (migration 000014). |
| `timezone` | VARCHAR(64) | default `'Asia/Jakarta'`. |
| `currency` | CHAR(3) | default `'IDR'`. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

References: none (root). Every other tenant table references this via a physical `store_id` FK.
Profil (`name`/`address`/`phone`/`logo_url`) dibaca-tulis oleh modul `settings`; siklus hidup
(`slug`/`status`) dibaca-tulis oleh modul `platform` — DUA pengecualian shared-kernel (lihat
`knowledge/DATABASE_GUIDE.md`). Profil disatukan dengan tabel `settings` dalam satu payload admin
`GET/PATCH /settings` dan `GET /pos/config`.

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

PK: `id`. Unique: `username` (GLOBAL — migration 000016; was `(store_id, username)`). Index: `store_id`.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `name` | VARCHAR(150) | NOT NULL. |
| `username` | VARCHAR(100) | NOT NULL, unique GLOBALLY (not just per-store) — fix for a login collision bug: the POS login endpoint (shared mobile APK, all tenants) carries no tenant identifier, so a per-store-only unique username let one tenant's staff collide with another's (silent lockout). Same fix already applied to `admin_users.username` in migration 000006. |
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
| `actor` | ENUM(`admin`,`staff`,`platform`) | which identity context (`platform` added in migration 000016). |
| `subject_id` | CHAR(26) | **primitive ID** → admin_users(id), staff(id), or platform_users(id) depending on `actor`. |
| `token_hash` | CHAR(64) | SHA-256 of the opaque refresh token, unique. |
| `expires_at` | DATETIME | NOT NULL. |
| `revoked_at` | DATETIME | nullable. |
| `created_at` | DATETIME | timestamp. |

---

## `platform_users` — platformuser / auth (migration 000016)

PK: `id`. Unique: `email`. Superadmin (platform operator) identity — the ONE table in this
schema with NO `store_id` column at all; not scoped to any tenant. `platformuser` owns CRUD
(create/deactivate-only/reset password — never hard-delete, §2.8/§2.9); `auth` separately keeps
its own narrow login-lookup queries against the same table (same split as `admin_users`/`staff`).

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `name` | VARCHAR(150) | NOT NULL. |
| `email` | VARCHAR(190) | NOT NULL, unique. |
| `password_hash` | VARCHAR(100) | bcrypt. |
| `status` | ENUM(`active`,`inactive`) | default `active`. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

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

## `payment_clients` — payment (PLAN.md §9.1.3 Part 2; §10.1.6 Part 3)

App registry — replaces the old `"sub_"` ref-prefix convention. PK: `id`. Unique: `app_id`.
Seeded (migration `000019`) with two `kind='internal'` rows: `ELKASIR-SELFORDER`,
`ELKASIR-SUBSCRIBE` — never hard-deleted, never deactivatable through the registry endpoints
(enforced by the `AND kind='external'` guard on the underlying UPDATE queries).

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `app_id` | VARCHAR(60) | Unique, human-readable (e.g. `ELKASIR-SELFORDER`, or `NAME-XXXX` for external apps). |
| `name` | VARCHAR(150) | Display name. |
| `secret_hash` | VARCHAR(100) | bcrypt hash. NULL for `kind='internal'` (no network hop to authenticate — §9.1.9). Verified on incoming `POST /auth/app/token` (§10.1.3). |
| `secret_enc` | VARBINARY(500) | Migration `000020`, Part 3. The SAME plaintext as `secret_hash`, but reversibly AES-256-GCM-encrypted (same key as `payment_gateway_config`'s secret columns) — needed ONLY to sign outbound webhook relays to `kind='external'` apps (bcrypt can't be reversed). NULL for `kind='internal'`. |
| `kind` | ENUM('internal','external') | `internal` = seeded, never created/deleted via the API. |
| `callback_url` | VARCHAR(500) | Nullable; only meaningful for `kind='external'` — the outbound webhook relay target (§10.1.10). |
| `status` | ENUM('active','inactive') | Checked live on every `ActorApp` request (§10.1.4) — no caching, deactivation revokes access to unexpired tokens immediately. |
| `created_at`/`updated_at` | DATETIME | |

## `payment_gateway_config` — payment (PLAN.md §9.1.2, Part 2)

Single logical row (enforced by application convention, not a DB constraint). Secret columns
are AES-256-GCM ciphertext, base64-encoded, keyed by `CONFIG_ENCRYPTION_KEY` (env) — never
plaintext, never returned to the browser (masked in `GetConfig`).

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `provider` | VARCHAR(20) | `tripay` \| `midtrans` \| `''` (simulation). |
| `sandbox` | TINYINT(1) | |
| `tripay_api_key_enc` / `tripay_private_key_enc` / `tripay_merchant_code_enc` | VARBINARY(500) | Encrypted; nullable. |
| `tripay_method` | VARCHAR(30) | Default channel code, e.g. `QRIS` — not a secret. |
| `midtrans_server_key_enc` | VARBINARY(500) | Encrypted; nullable. |
| `created_at`/`updated_at` | DATETIME | |

## `payment_charge_apps` — payment (PLAN.md §9.1.4 Part 2; §10.2 EB2 Part 3)

Thin dispatch index — NOT a business ledger. PK: `order_ref`. Written by
`CreateCharge`/`CreateChannelCharge`, read once per incoming webhook to resolve which
registered `app_id` (and therefore which in-process consumer) a callback belongs to. Also
doubles (Part 3) as the ownership + ref-translation lookup behind the external
`GET /external/payments/charges/{orderRef}/status` route — the PK uniqueness on `order_ref` is
also what turns a retried `POST /external/payments/charges` into a `409 Conflict` (§10.1.9,
via the existing `db.IsDuplicate` helper — no new idempotency table).

| Column | Type | Notes |
|--------|------|-------|
| `order_ref` | VARCHAR(191) | PK — the ref passed to `CreateCharge`, echoed back by the gateway on webhook. |
| `app_id` | VARCHAR(60) | Not a DB FK to `payment_clients.app_id` (cross-module-style primitive reference, same convention as everywhere else in this schema). |
| `provider_ref` | VARCHAR(191) | Migration `000021`, Part 3. Nullable. The gateway's own reference for this charge — lets the external status endpoint translate a caller's `orderRef` into the `provider_ref` needed by `CheckStatus`, without exposing gateway internals to the caller. |
| `created_at` | DATETIME | |

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

PK: `id`. Unique: `(store_id, code)`. `code` encodes the self-order QR
(`/order/<store-slug>/<code>` — the tenant slug is REQUIRED in the URL since `code` alone is only
unique per-store; the public self-order entry point joins to `stores.slug` to resolve the
tenant — see `FindTableByStoreSlugAndCode` and `knowledge/DATABASE_GUIDE.md` §3).

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

PK: `id`. Index: `(store_id, created_at)`. `status` enum's 4 values are all meaningfully used by
the claim→complete flow since migration 000017 (§2.7): `pending` → (Klaim) → `processing` →
(Tandai Sukses) → `success`, or → (Tolak, from either `pending` or `processing`) → `failed`.

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
| `processed_by` | CHAR(26) | nullable (migration 000017). **primitive ID** → platform_users(id) — set at claim time (`Claim`), carried through to the outcome (`MarkSuccess`/`MarkRejected`); never a FK, same "Bebas dari Penjara FK" convention as `requested_by`. |
| `claimed_at` | DATETIME | nullable (migration 000017) — set when a superadmin claims (`pending`→`processing`). |
| `processed_at` | DATETIME | nullable (migration 000017) — set on the final outcome (`success` or `failed`). |
| `rejected_reason` | VARCHAR(255) | nullable (migration 000017) — required input for `Tolak`, shown to the tenant. |
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

## `payments` — selforder

PK: `id`. Indexes: `self_order_id`, `provider_ref`. Gateway (Tripay/Midtrans) payment history/
reconciliation for CUSTOMER self-order payments — **not** tenant billing (see `subscription_invoices`
below, a separate table owned by a separate module, even though both go through the same `payment`
gateway module). Originally owned by `payment`; moved to `selforder` so the two business domains'
money never share a table — see `knowledge/MODULE_MAP.md` §"One shared payment webhook, two consumers".

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `self_order_id` | CHAR(26) | NOT NULL. Now an **intra-module** reference (both tables owned by `selforder`) — no FK constraint exists (dropped in 000005, back when `payment` still owned this table), one could be added back but isn't required. |
| `provider` | ENUM(`xendit`,`midtrans`,`tripay`) | default `tripay` (migration 000008; older values kept for back-compat). |
| `provider_ref` | VARCHAR(190) | nullable (external ref). |
| `method` | ENUM(`qris`) | default `qris`. |
| `amount` | BIGINT | default 0. |
| `status` | ENUM(`pending`,`paid`,`expired`,`failed`) | default `pending`. |
| `raw_payload` | LONGTEXT | nullable (raw provider payload). |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## `subscription_plans` — subscription

PK: `id`. Unique: `code`. Reference/catalog data (seeded via `bootstrap.Seed`), not tenant data.
Includes one hidden row, `code='legacy-grandfather'` (`is_active=0`, seeded by migration
`000018_subscription_legacy_backfill`) — never shown in the tenant-facing plan picker, exists
only as the backfill target for pre-existing tenants (see `store_subscriptions` above).

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `code` | VARCHAR(40) | NOT NULL, unique (e.g. `basic`, `pro`). |
| `name` | VARCHAR(100) | NOT NULL. |
| `price` | BIGINT | NOT NULL — billed amount (rupiah). |
| `period_days` | INT | default 30 — billing cycle length. |
| `is_active` | TINYINT(1) | default 1. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## `store_subscriptions` — subscription

PK: `id`. Unique: `store_id` (one row per store). Index: `plan_id`. Current billing status of a
tenant. "Has an active package" (§2.15, gates `ActorAdmin`/`ActorStaff` access) = a row with
`status='active'` AND `current_period_end >= NOW()`, computed live on every gated
login/request — no cron ever flips `status`. Migration 000018 backfills every pre-existing
tenant onto a hidden `legacy-grandfather` plan (see `subscription_plans` below) so this gate
doesn't retroactively lock out tenants who predate it.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `plan_id` | CHAR(26) | **physical FK → subscription_plans(id)** (intra-module). |
| `status` | ENUM(`trial`,`active`,`past_due`,`expired`,`canceled`) | default `trial`. |
| `current_period_start` / `current_period_end` | DATETIME | nullable — set/extended on invoice payment. |
| `created_at` / `updated_at` | DATETIME | timestamps. |

---

## `subscription_invoices` — subscription

PK: `id`. Indexes: `store_id`, `provider_ref`. This module's OWN gateway payment ledger — the
tenant-billing analogue of selforder's `payments`, but a **separate table** so the two domains'
money is never mixed.

| Column | Type | Notes |
|--------|------|-------|
| `id` | CHAR(26) | PK — also embedded (prefixed `sub_`) as the gateway order ref, so the shared payment webhook can route the callback back here. |
| `store_id` | CHAR(26) | **physical FK → stores(id)** (CASCADE). |
| `plan_id` | CHAR(26) | **physical FK → subscription_plans(id)** (intra-module). |
| `amount` | BIGINT | default 0 — snapshot of the plan price at checkout time. |
| `status` | ENUM(`pending`,`paid`,`expired`,`failed`) | default `pending`. |
| `provider` | ENUM(`tripay`,`midtrans`) | NOT NULL — from `paymentclient.Charge.Provider`. |
| `provider_ref` | VARCHAR(190) | nullable (external ref). |
| `period_start` / `period_end` | DATETIME | nullable — filled in on payment confirmation (not at checkout time, since payment may land days later). |
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
| `withdrawals.processed_by` | `platform_users.id` | platformuser (added migration 000017 — never had a physical FK) |
| `self_orders.table_id` | `dining_tables.id` | table |
| `self_orders.transaction_id` | `transactions.id` | transaction |
| `self_order_items.product_id` | `products.id` | product |

`payments.self_order_id` is **not** listed above — it's an intra-module reference (both tables
owned by `selforder`), not a cross-module one; see the `payments` section.

**Retained physical FKs:** every `store_id → stores(id)` (tenant key, CASCADE), the intra-module
links `products.category_id → product_categories(id)` (SET NULL),
`transaction_items.transaction_id → transactions(id)` (CASCADE),
`self_order_items.self_order_id → self_orders(id)` (CASCADE), and the `subscription` module's own
`store_subscriptions.plan_id` / `subscription_invoices.plan_id → subscription_plans(id)`.
