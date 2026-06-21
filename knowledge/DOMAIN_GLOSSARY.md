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
unit of **multi-tenancy**.

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
Moving funds out of the store's balance to a bank account. Records `amount`, `bank`,
`account`, account `holder`, a `status`, an optional `reference`, and the requester.
It reduces the cash available and is reflected in shift totals. It is a recorded payout
intent with status — not a live bank-transfer execution.

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

### Admin user vs. Staff
Two distinct kinds of accounts, authenticated as two distinct **actor types**:

- **Admin user** — a web dashboard user (owner/manager type). Roles: `owner`, `admin`,
  `manager`, `viewer`. Manages catalog, tables, people, and reads reports. Identified
  by `email`.
- **Staff** — a POS operator who logs into the mobile cashier app. Roles: `cashier`,
  `supervisor`. Opens/closes shifts and records counter sales. Identified by
  `username`.

These are separate populations: an admin user is not a staff member and vice versa.

### Actor
The category of identity attached to an authenticated session (`actor` on the auth
user). Elkasir has two actor types — **admin** (web dashboard) and **staff** (POS) —
which the API uses to decide which endpoints and data a token may access.

### Role
The permission level within an actor type. For **admin users**: `owner` > `admin` >
`manager` > `viewer`. For **staff**: `supervisor`, `cashier`. Roles gate what actions a
user may perform (e.g. who can approve a shift close).

### QRIS (Quick Response Code Indonesian Standard)
Indonesia's unified national QR-payment standard. A single QR code can be scanned and
paid from any compliant mobile banking or e-wallet app. In Elkasir, QRIS is the digital
payment method for both self-orders and counter sales.

### Payment gateway (Tripay / Midtrans)
The third-party QRIS provider Elkasir integrates with to generate payments and receive
payment-status confirmations via callback/webhook. The `payment` module is **provider-agnostic**:
exactly one provider is active, chosen by `PAYMENT_PROVIDER` (`tripay` | `midtrans`), with
`PAYMENT_ENV` selecting sandbox vs production.

- **Tripay** (active) — Closed Payment QRIS; charge via `transaction/create` (Bearer API Key,
  `signature = HMAC-SHA256(merchantCode + merchantRef + amount, privateKey)`). Callbacks are
  verified by the `X-Callback-Signature` header (HMAC-SHA256 of the raw body); status `PAID` means paid.
- **Midtrans** (selectable) — Core API QRIS; charge via `/v2/charge`. Webhooks are verified by
  `signature_key` (`SHA512(order_id + status_code + gross_amount + ServerKey)`); status `settlement` means paid.

A `simulated` mode (active provider's credentials empty) allows development without live charges.

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
