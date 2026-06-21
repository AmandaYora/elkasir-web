# Elkasir — Project Brief

## What Elkasir is

**Elkasir** is a multi-tenant point-of-sale (POS) system built for food & beverage
(F&B) businesses such as cafés, restaurants, and warungs. It bundles the day-to-day
operations of a small-to-medium F&B outlet into one product: managing the menu,
taking orders (both at the counter and directly from the customer's phone),
recording sales, handling the cash drawer across staff shifts, accepting digital
payments, and giving the owner a clear picture of how the business is doing.

A single Go API powers two distinct web frontends, while a separate Flutter mobile
app handles the cashier-facing POS.

## Why it exists

Small F&B operators usually juggle several disconnected tools — a spreadsheet for
the menu, a cash box with a paper log, a separate QR-payment app, and guesswork at
the end of the day for reconciliation. Elkasir replaces that with one coherent
system that:

- Keeps the **menu and pricing** in one place, instantly reflected wherever orders
  are taken.
- Lets customers **self-order by scanning a table QR code**, reducing queue time and
  cashier load.
- Records **every sale** with its source (cashier vs. self-order) and payment method
  (cash vs. QRIS), so revenue is auditable.
- Enforces **shift discipline** — opening float, expected vs. actual cash, and
  variance — so cash shortfalls are visible, not silent.
- Surfaces **reports and analytics** (revenue, top products, payment mix, staff
  performance) so owners make decisions on data, not feel.

## Who uses it

| Role | Surface | What they do |
| --- | --- | --- |
| **Owners / Managers** | Web admin dashboard (React 19 SPA) | Manage products, categories, tables, staff, and admin users; monitor incoming self-orders; review shifts, cash movements, and withdrawals; read reports and analytics. |
| **Customers** | Self-order pages (public web) | Scan a table QR code, browse the menu, place an order, and pay via QRIS or cash. No login required. |
| **Cashiers** | Mobile POS app `elkasir_pos` (Flutter, **separate repo**) | Open/close shifts, take counter orders, accept payments, manage the cash drawer. Consumes the same API; not part of this repository. |

All three surfaces talk to the **same Go API** under `/api/v1`. The web admin SPA and
the customer self-order pages live in this repository (`apps/web`); the API lives in
`apps/api`; the Flutter cashier app is a separate project that only consumes the API.

## Multi-tenancy

Elkasir is **multi-tenant**: every business is a *store*, and all domain data —
products, tables, staff, transactions, shifts, cash records — is scoped to a
`store_id`. One deployment can serve many stores, each seeing only its own data.

## How it is deployed (one-container model)

Elkasir ships as **one container**. At build time the React SPA is compiled to static
assets and **embedded directly into the Go binary**. At runtime, a single process:

- serves the **SPA at the root path** (admin dashboard + self-order pages), and
- serves the **API under `/api/v1`**.

There is no separate web server, no separate API server, and no frontend container —
just one binary doing both. **MySQL runs at the host/OS level** (not containerized);
the container connects to it over the host network. This keeps the operational
footprint tiny: one app container plus a host database, fronted by a reverse proxy
(e.g. Caddy or nginx) for TLS.

### At a glance

| Aspect | Choice |
| --- | --- |
| Frontends | Web admin (React 19 SPA) + customer self-order pages (same SPA build) |
| Cashier POS | Flutter app `elkasir_pos` (separate repo) |
| Backend | Go modular monolith (go-chi, MySQL via sqlc + golang-migrate, ULID ids) |
| Auth | JWT, two actor types: admin web users and POS staff |
| Payments | QRIS gateway — Tripay (active) or Midtrans, selectable via env |
| Tenancy | Per-`store_id` scoping |
| Deploy | One container (SPA embedded in Go binary); MySQL on host |
| API base path | `/api/v1` |

## Repository shape (target)

```
apps/
  web/    # React 19 SPA: admin dashboard + self-order pages
  api/    # Go modular monolith; serves SPA at root and API under /api/v1
packages/
  api-contract/   # OpenAPI contract (source of truth for types)
  shared/         # domain-agnostic TypeScript utilities
knowledge/        # project documentation (this folder)
```
