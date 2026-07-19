# Elkasir — Domain Glossary

Shared terminology for Elkasir. These definitions are the canonical meaning of each
term across the API, the web admin, the self-order pages, and the cashier POS. Where a
term maps to a concrete field/value in the codebase, that value is shown in
`monospace`.

---

### Store
A single F&B business (tenant). Every domain record — products, tables, staff,
transactions, shifts, cash records — is scoped to a `store_id`. One Elkasir
deployment can host many stores, and each store sees only its own data. This is the
unit of **multi-tenancy**. Owns two lifecycle fields set by the `platform` module (a
shared-kernel exception, same class as `settings`' profile columns): `slug` (unique,
stable identity used in the public self-order URL — `/order/{slug}/{tableCode}` —
since a table `code` alone is only unique per-store) and `status` (`active` /
`suspended` — a superadmin action that immediately blocks that tenant's every
subsequent request, web and mobile, even on an already-issued token).

### Product
A sellable menu item. Carries a name, `sku`, optional `categoryId`, selling `price`,
`cost`, `stock`, a `status` (`active` / `inactive`), and an optional image. Only
`active` products appear on the customer self-order menu and the cashier POS menu.

### SKU (Stock Keeping Unit)
A short, store-unique code identifying a product (e.g. `COF-LATTE-01`). Used to
reference and disambiguate items independently of their display name.

### Category
A grouping of products (e.g. "Coffee", "Snacks"). Has a `name`, a `sortOrder` that
controls display order, and a derived `productCount`. Categories organize the menu for
both staff and customers.

### Table
A physical seating spot in the venue. Has a `code`, `name`, `area`, `seats`, and a
`status` (`active` / `inactive`). Each table is the QR entry point for self-ordering —
scanning its QR opens that table's public menu.

### Self-order
An order placed directly by a **customer** from their own phone after scanning a
table's QR code — no staff or login involved. It moves through statuses
`placed` → `preparing` → `completed`, has its own `paymentStatus`
(`pending` / `paid` / `unpaid` / `expired` / `failed`), a chosen `paymentMethod`
(`qris` / `cash`), and once settled it produces a **transaction**.

### Claim code
A short code issued for a self-order (`claimCode`) so the customer and staff can
identify and match the order (e.g. to hand over food or confirm payment). It is a
human-friendly reference, distinct from the internal order ID.

### Shift
A continuous period during which one staff member operates the cash drawer. Has a
`status` of `open` or `closed` and accumulates the financial activity of that period.

- **Open / Close** — a shift is *opened* with an initial cash float and *closed* when
  the staff member counts the drawer. Closing records who approved it
  (`closeApprovedBy`) and timestamps (`openedAt`, `closedAt`).
- **Initial cash** — the starting float in the drawer (`initialCash`).
- **Expected cash** — what the drawer *should* contain at close, computed from initial
  cash plus cash sales, additional capital, and adjustments, minus expenses and
  withdrawals (`expectedCash`).
- **Actual cash** — the cash physically counted at close (`actualCash`).
- **Variance** — `actualCash − expectedCash`. A negative variance is a shortfall, a
  positive one is an overage (`variance`). Variance makes cash discrepancies visible
  and accountable.

### Cash movement
A recorded change to the cash drawer that is **not** a sale. Each has a `type`, an
`amount`, optional `notes`, and may be tied to a shift, with a creator and approver.
The three types:

- **Capital** (`capital`) — cash added into the drawer (e.g. an extra float / cash
  injection by the owner).
- **Expense** (`expense`) — cash paid out of the drawer for operating costs (e.g.
  buying supplies).
- **Adjustment** (`adjustment`) — a deliberate correction to the recorded cash to fix
  a known discrepancy, logged transparently rather than silently.

### Withdrawal
Moving a tenant's **self-order QRIS earnings** — sitting in Elkasir's shared gateway wallet, not
the physical cash drawer — out to the tenant's own bank account. Records `amount`, `bank`,
`account`, account `holder`, a `status`, an optional `reference`, the requester, and (once
claimed) `processedBy`/`claimedAt`/`processedAt`/`rejectedReason`. The actual transfer always
happens outside the system (a superadmin's own banking app) — not a live bank-transfer API
integration.

Status moves through a **two-step claim → complete flow**, not a single action, because more
than one superadmin can exist and there's no database lock over an outside bank transfer:
`pending` **[any superadmin: Claim]** `processing` **[only the claimant: Mark Success]**
`success`, or `pending`/`processing` **[any superadmin: Reject]** `failed`. See
**AvailableBalance** and **Claimable balance** below for the two balance formulas that gate
this.

### AvailableBalance
The reconciliation-accurate balance shown everywhere (Ringkasan, Revenue Tenant, the tenant's own
page) = a tenant's self-order QRIS revenue minus the sum of that tenant's `success` withdrawals.
Money hasn't left the gateway until a withdrawal is `success`, so this is the figure that should
sum against the real Tripay/Midtrans balance.

### Claimable balance
An internal-only check (never shown as its own UI figure) used by both a withdrawal `Create` and
a `Claim` action = **AvailableBalance** minus that tenant's currently-`processing` withdrawals.
Narrower than AvailableBalance on purpose: money already claimed by a superadmin (about to be
transferred) is still physically in the gateway, but "spoken for" — a second request/claim must
not also count it as free. A merely-`pending` (unclaimed) request never reserves balance either
way.

### Transaction
A completed sale. Records a `code`, line `items`, money fields (`subtotal`,
`discount`, `tax`, `total`, `amountReceived`, `changeAmount`), a `status`, and links to
the relevant `shiftId`, `tableId`, `selfOrderId`, and `cashierId`. Two attributes
classify every transaction:

- **Source** (`source`) — where it originated:
  - `cashier` — entered at the counter via the mobile POS by a staff member.
  - `self_order` — generated from a customer self-order.
- **Payment method** (`paymentMethod`):
  - `cash` — paid in physical cash; `amountReceived` and `changeAmount` apply.
  - `qris` — paid digitally via QRIS (through the active gateway: Tripay/Midtrans).

### Admin user vs. Staff vs. Platform user vs. App
Four distinct kinds of accounts, authenticated as four distinct **actor types** — never mixed;
separate identity domains, separate token storage on the frontend:

- **Admin user** (`actor=admin`) — a web dashboard user (owner/manager type), scoped to one
  `store_id`. Roles: `owner`, `admin`, `manager`, `viewer`. Manages catalog, tables, people,
  reads reports, manages the tenant's own subscription. Identified by `email`.
- **Staff** (`actor=staff`) — a POS operator who logs into the mobile cashier app, scoped to one
  `store_id`. Roles: `cashier`, `supervisor`. Opens/closes shifts and records counter sales.
  Identified by `username`.
- **Platform user** (`actor=platform`) — a **superadmin**, Elkasir's own operator. No `store_id`
  at all — the only identity in the whole schema not tied to a tenant. Logs into Konsol Platform.
  No role tiers (every platform user has the same authority). Identified by `email`.
- **App** (`actor=app`) — a registered external payment-API caller (`payment_clients`, Part 3).
  No `store_id`, no role. Authenticates via client-credentials (`POST /auth/app/token`), gets a
  short-lived access token and **no refresh token** (re-exchanges instead). `Principal.SubjectID`
  holds the `payment_clients.id` row.

These are separate populations: an admin user is not a staff member, a platform user is not tied
to any store, and an app has no human behind it at all.

### Actor
The category of identity attached to an authenticated session (`actor` on the auth
user/principal). Elkasir has **four** actor types — `admin` (web dashboard), `staff` (POS),
`platform` (superadmin/Konsol Platform), `app` (external payment API) — which the API uses to
decide which endpoints and data a token may access. `RequireActor(...)` middleware rejects `403`
if the principal's actor doesn't match what a route requires.

### Role
The permission level within an actor type. For **admin users**: `owner` > `admin` >
`manager` > `viewer`. For **staff**: `supervisor`, `cashier`. `platform` and `app` actors carry
no role at all (`platform` has no tiers; `app` has none to speak of). Roles gate what actions a
user may perform (e.g. who can approve a shift close, or checkout/upgrade a subscription —
owner-only).

### QRIS (Quick Response Code Indonesian Standard)
Indonesia's unified national QR-payment standard. A single QR code can be scanned and
paid from any compliant mobile banking or e-wallet app. In Elkasir, QRIS is the digital
payment method for both self-orders and counter sales.

### Payment gateway (Tripay / Midtrans)
The third-party QRIS/VA provider Elkasir integrates with to generate payments and receive
payment-status confirmations via callback/webhook. The `payment` module is **provider-agnostic**
and, since Part 2, **DB-configured**: exactly one provider is active at a time ("one wallet" —
never per-app credentials), edited from Konsol Platform (Konfigurasi Pembayaran) — not
`PAYMENT_PROVIDER`/`.env`. Credentials are stored AES-256-GCM-encrypted (`CONFIG_ENCRYPTION_KEY`
derives the key). `.env`'s `TRIPAY_*`/`MIDTRANS_*`/`PAYMENT_PROVIDER`/`PAYMENT_ENV` are read only
once, on first boot after this feature shipped, to migrate a deployment's existing config into
the DB — never read again after that.

- **Tripay** — Closed Payment QRIS/VA; charge via `transaction/create` (Bearer API Key,
  `signature = HMAC-SHA256(merchantCode + merchantRef + amount, privateKey)`). Callbacks are
  verified by the `X-Callback-Signature` header (HMAC-SHA256 of the raw body); status `PAID` means paid.
- **Midtrans** — Core API QRIS; charge via `/v2/charge`. Webhooks are verified by
  `signature_key` (`SHA512(order_id + status_code + gross_amount + ServerKey)`); status `settlement` means paid.

A `simulated` mode (active provider's credentials empty) allows development without live charges.

### App registry (`payment_clients`)
The set of internal, in-process Go consumers allowed to create charges through the one shared
Tripay/Midtrans wallet (Part 2). Each row has an `app_id` (e.g. `ELKASIR-SELFORDER`,
`ELKASIR-SUBSCRIBE`) and a `status`. Replaces an earlier ad-hoc `"sub_"` order-ref-prefix
convention used to guess which consumer owned an incoming webhook. Part 3's `kind=external`
capability (letting a separate SaaS product call in over HTTP through this same registry) was
removed (PLAN.md §11) — Elkasir is now a client of ElProof instead, not a provider to other apps.

### ElProof
A separate, standalone payment-gateway product (`elproof.elcodelabs.com`) — the same "one wallet,
many consumers" pattern Elkasir itself built, spun out as its own product. Elkasir is registered
there as external app `Elkasir-Billing`, used ONLY for subscription billing (`AppSubscribe`) —
`selforder` is unaffected and still uses Elkasir's own Tripay/Midtrans wallet. See PLAN.md §11.

### Konsol Platform
The superadmin-facing surface (`/platform/*`) — same visual shell as the tenant admin dashboard,
distinguished only by a sidebar subtitle. Covers tenant lifecycle, revenue reconciliation, plan
catalog, withdrawal claim/complete processing, superadmin user management, and payment gateway
config + app registry. See **Superadmin**.

### Superadmin
Elkasir's own operator role (`platform_users`, `actor=platform`). Not tied to any `store_id` —
the only identity in the schema that isn't. No role tiers (every superadmin has equal authority);
never hard-deleted, only deactivated (an active audit trail — e.g. who claimed a withdrawal —
must never dangle after an account is removed). Uses Konsol Platform.

### Plan (subscription plan)
Reference/catalog data for **tenant billing to Elkasir** (a separate business domain from
self-order — the store is the payer here, Elkasir is the payee). Has a `code` (stable, immutable
after creation), `name`, `price`, `periodDays`, `isActive` (shown in the tenant-facing picker or
not), and `renewalOnly` (see below). Managed from Konsol Platform.

### Renewal-only plan
A plan (`renewalOnly=true`) that a subscriber may only ever **renew** — never switch to it from
another plan, nor switch away from it once assigned. Enforced server-side
(`subscription/application.Service.validatePlanSwitch`), not just hidden in the UI. Used for
"Premium Contributor" (see below) but is a general plan property, not hardcoded to one plan.

### Premium Contributor
The specific renewal-only plan (`code=premium-contributor`, Rp1.700.000/year) that pre-existing
tenants were backfilled onto when subscription billing was introduced — hidden from the normal
plan picker (`isActive=false`, only ever assignable via that one-time migration), with an initial
365-day grace period before the first real renewal charge.

### Rincian biaya (pricing breakdown)
Setiap transaksi/self-order memecah pembayaran menjadi:
- **Subtotal** — total harga barang (penjualan).
- **Service (biaya layanan)** — `2% × Subtotal`, **dibulatkan ke atas** (sisa thd ribuan ≤500→x.500,
  >500→x.000 berikutnya). Berlaku untuk **semua** transaksi (cash/QRIS, kasir/self-order). Margin merchant.
- **Gateway fee (biaya gateway)** — biaya provider QRIS (Tripay live / Midtrans), **hanya untuk QRIS**;
  0 untuk kasir/cash. Pass-through ke gateway.
- **Layanan** — baris yang ditampilkan ke pelanggan = `Service + Gateway fee`.
- **PPN (pajak)** — `taxPercent% × Subtotal` bila `taxEnabled` (di menu Pengaturan); default mati.
- **Total** — `Subtotal − Diskon + Service + Gateway fee + PPN`. Inilah yang ditagih/dibayar.

Pemisahan keuangan (laporan) memakai 3 bucket: **Penjualan** (subtotal−diskon), **Layanan**
(service + gateway), **Pajak** (PPN) — ketiganya berjumlah = revenue (SUM total).

### Pengaturan (Settings)
Konfigurasi per-toko milik modul `settings` (menu admin "Pengaturan"): ambang kontrol (diskon
maks, biaya operasional, toleransi selisih kas), flag fitur (self-order, QRIS), dan **pajak &
layanan** (`taxEnabled`, `taxPercent`, `servicePercent`). Modul lain membaca via `settingsclient`.
