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
| **Owners / Managers** | Web admin dashboard (React 19 SPA) | Manage products, categories, tables, staff, and admin users; monitor incoming self-orders; review shifts, cash movements, and withdrawals; read reports and analytics; manage the tenant's own **subscription** to Elkasir (Langganan page). |
| **Customers** | Self-order pages (public web) | Scan a table QR code (URL includes the tenant's `slug`), browse the menu, place an order, and pay via QRIS or cash. No login required. |
| **Cashiers & Supervisors** | Mobile POS app `elkasir_pos` (Flutter, **sibling project `../elkasir_mobile`**) | Open/close shifts, take counter orders, accept cash/QRIS, manage the cash drawer, redeem pay-at-cashier self-orders. **Supervisors** additionally approve over-cap discounts, voids, and over-tolerance shift closes (PIN verified server-side). Consumes the same API; lives in a separate project (see `../elkasir_mobile/CLAUDE.md`). |
| **Superadmin** | **Konsol Platform** (`/platform/*`, same React 19 SPA, separate login/session domain) | Elkasir's own operator role — onboards new tenants, suspends/reactivates them, manages the subscription plan catalog, claims/completes tenant cash-withdrawal requests, reconciles platform-wide revenue, manages other superadmin accounts, and configures payment gateway credentials (Tripay/Midtrans for self-order, plus ElProof for subscription billing — PLAN.md §11). Not tied to any `store_id`. |

Elkasir was previously also a payment-gateway-as-a-service **provider** for other SaaS products
(Part 3, PLAN.md §10) — that capability was removed (§11). Elkasir is now itself a **client** of a
separate product, ElProof, for its own subscription billing only.

All surfaces talk to the **same Go API** under `/api/v1`. The web admin SPA, the
Konsol Platform SPA, and the customer self-order pages live in this repository
(`apps/web`, same build — routed by path, not a separate bundle); the API lives in
`apps/api`; the Flutter cashier/supervisor app is the **sibling project
`../elkasir_mobile`** (`elkasir_pos`) that only consumes this API — see its gateway
`../elkasir_mobile/CLAUDE.md`.

## Multi-tenancy

Elkasir is **multi-tenant**: every business is a *store*, and all domain data —
products, tables, staff, transactions, shifts, cash records — is scoped to a
`store_id`. One deployment can serve many stores, each seeing only its own data.

A tenant's access is gated two independent ways, both enforced live on every
authenticated request (no caching, no cron):

- **Suspension** (`stores.status`) — a superadmin can suspend a tenant from Konsol
  Platform; every subsequent request (web admin *and* mobile POS, since both go
  through this same API) is rejected immediately, even on an already-issued token.
- **Subscription status** (§ below) — a tenant with no active paid plan is
  restricted to the Langganan (subscription) page only (owner/admin) or fully
  blocked (POS staff), independent of suspension.

## Elkasir's own business model: subscription billing

Elkasir charges **tenants** (the F&B businesses) a recurring fee to use the
platform — a second, separate revenue flow from the QRIS payments tenants collect
from *their own* customers. Both flows settle through the **same shared
Tripay/Midtrans merchant account** ("one wallet"), so Konsol Platform's revenue
dashboard exists specifically to reconcile the two: subscription revenue (Elkasir's
own income) plus tenants' undisbursed self-order balances should equal the real
gateway balance. A tenant checks out a plan from its own Langganan page; payment
confirmation unlocks the platform the same way a customer's self-order QRIS payment
unlocks their order. New tenants get no free trial — pre-existing tenants (from
before this billing model existed) were grandfathered onto a locked, renewal-only
plan with one free year, not exempted indefinitely.

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
| Frontends | Web admin (React 19 SPA) + Konsol Platform (superadmin) + customer self-order pages (same SPA build) |
| Cashier POS | Flutter app `elkasir_pos` (separate repo) |
| Backend | Go modular monolith (go-chi, MySQL via sqlc + golang-migrate, ULID ids) |
| Auth | JWT, four actor types: admin (web owner/manager), staff (POS), platform (superadmin), app (external payment API caller) |
| Payments | QRIS/VA gateway — Tripay or Midtrans, exactly one active, **DB-configured** (superadmin sets it in Konsol Platform; `.env` only bootstraps it once on first boot) |
| Tenancy | Per-`store_id` scoping; suspension + subscription-status gates enforced live on every request |
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
