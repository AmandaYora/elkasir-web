# Elkasir — Product Requirements

This document captures the features and user stories of Elkasir, grouped by
functional area. Roles referenced:

- **Owner / Manager** — uses the web admin dashboard (`apps/web`).
- **Cashier (Staff)** — uses the separate Flutter POS app `elkasir_pos`.
- **Customer** — uses the public self-order pages (no account).
- **Superadmin** — Elkasir's own operator role, uses **Konsol Platform** (`/platform/*`, same
  `apps/web` build, separate login/session domain). Not tied to any store.
- **External integrator** — a separate SaaS product's backend, registered as a `kind=external`
  app; calls the external payment API server-to-server, no UI.

All data is scoped to a **store** (multi-tenant). The API is served under `/api/v1`
with standard response envelopes (`{success, message, data}`, paginated variants with
`meta`, and `{success, message, errors}` for errors).

---

## 1. Catalog — Products & Categories

Backend modules: `product`, `category`.

### Scope
- Products with: name, SKU, category, selling price, cost, stock, status
  (`active` / `inactive`), optional image.
- Categories with: name, sort order, and a derived product count.
- Active products feed both the cashier POS menu and the customer self-order menu.

### User stories
- As an **owner**, I want to create, edit, and deactivate products so that my menu
  reflects what is actually for sale.
- As an **owner**, I want each product to have an SKU, a selling price, and a cost so
  that I can identify items uniquely and understand margins.
- As an **owner**, I want to set a product to `inactive` so that it disappears from
  the customer menu without losing its history.
- As an **owner**, I want to group products into categories and order those
  categories so that the menu is easy to navigate.
- As a **customer**, I want to browse the menu grouped by category so that I can find
  items quickly.

### Non-goals
- No multi-variant/modifier engine (e.g. size/topping matrices) in this scope.
- Stock is tracked as a simple number; no warehouse, batch, or supplier management.

---

## 2. Tables & QR Self-Order

Backend modules: `table`, `selforder`.

### Scope
- Tables with: code, name, area, seat count, status (`active` / `inactive`).
- Public menu fetched by table code (`PublicMenu`).
- Self-order lifecycle: `placed` → `preparing` → `completed`.
- Self-order payment status: `pending`, `paid`, `unpaid`, `expired`, `failed`.
- A **claim code** is issued so the customer/staff can identify the order.
- Self-orders may pay by `qris` or `cash` and ultimately produce a transaction.

### User stories
- As an **owner**, I want to define tables (code, area, seats) so that each physical
  table maps to a QR entry point.
- As a **customer**, I want to scan a table's QR code and see that table's menu so
  that I can order without waiting for staff.
- As a **customer**, I want to add items, leave a note, and place an order so that
  the kitchen knows what to prepare.
- As a **customer**, I want to pay by QRIS (or choose cash) at checkout so that I can
  complete my order from my phone.
- As a **customer**, I want a claim code and a way to see my order status so that I
  know whether it is placed, preparing, or completed.
- As an **owner/manager**, I want to see incoming self-orders so that staff can
  fulfil them and reconcile payment.

### Non-goals
- No table reservation/booking system.
- No live kitchen-display-system (KDS) beyond order status; status is updated, not
  streamed as a dedicated KDS surface.

---

## 3. Staff & Admin-User Management

Backend modules: `staff`, `adminuser`, `auth`.

### Scope
- **Staff (POS)**: name, username, optional email, role (`cashier` / `supervisor`),
  status (`active` / `inactive`). Authenticate to the mobile POS.
- **Admin users (web)**: name, email, role (`owner` / `admin` / `manager` /
  `viewer`), status, last-active timestamp. Authenticate to the web dashboard.
- Two distinct **actor types** in auth: admin web users and POS staff.

### User stories
- As an **owner**, I want to create POS staff accounts with cashier or supervisor
  roles so that the right people can operate the till.
- As an **owner**, I want to deactivate a staff account so that a former employee can
  no longer log in, without deleting their sales history.
- As an **owner**, I want to invite other admin users (admin/manager/viewer) so that
  responsibilities can be delegated with appropriate access.
- As a **manager**, I want a viewer-level admin role to exist so that some people can
  read reports without changing data.
- As any **user**, I want to log in and have my session secured by JWT so that access
  is authenticated and scoped to my store and actor type.

### Non-goals
- No fine-grained custom permission editor; roles are a fixed set.
- No role tiers *within* the superadmin population (`platform_users`) — see §9. A cross-store
  global superadmin role **does exist** (Konsol Platform, §9) — this section's scope is only the
  per-store `adminuser`/`staff` populations.

---

## 4. Sales & Transactions

Backend module: `transaction`.

### Scope
- A transaction records: code, source (`cashier` / `self_order`), payment method
  (`cash` / `qris`), status, line items, subtotal, discount, tax, total,
  amount received, change, and optional links to shift, table, self-order, and
  cashier.
- Each line item records product name, category, price, quantity, line total, note.
- Transaction creation is **atomic** and orchestrated across product (stock), shift
  (totals), and sales modules via contract clients + Unit-of-Work.

### User stories
- As a **cashier**, I want to record a sale with its items, payment method, and
  amount received so that change is computed and the sale is logged.
- As a **cashier**, I want a cash sale to compute change from amount received so that
  I hand back the correct amount.
- As an **owner**, I want every transaction tagged by source (cashier vs.
  self-order) so that I can see where revenue originates.
- As an **owner**, I want each transaction tied to the shift in which it occurred so
  that shift totals reconcile.
- As an **owner**, I want to browse and inspect past transactions so that I can audit
  sales.

### Non-goals
- No refund/void workflow defined in this scope (status field exists but a full
  return engine is out of scope).
- No split-payment across multiple methods on one transaction.

---

## 5. Shifts & Cash Management

Backend modules: `shift`, `cashmovement`.

### Scope
- A **shift** belongs to a staff member and has status `open` / `closed`.
- Tracks: initial cash (float), cash sales, QRIS sales, additional capital,
  expenses, withdrawals, adjustments, drawer-open count, expected cash, actual cash,
  variance, and who approved the close.
- **Cash movements** of type `capital`, `expense`, or `adjustment`, each optionally
  tied to a shift, with amount, notes, creator, and approver.

### User stories
- As a **cashier**, I want to open a shift with an initial cash float so that the
  drawer's starting state is recorded.
- As a **cashier**, I want to close my shift by entering the actual counted cash so
  that the system computes variance against expected cash.
- As a **supervisor**, I want shift closes to record who approved them so that
  variance is accountable.
- As a **cashier/supervisor**, I want to log capital injections and expenses against
  a shift so that the drawer math stays correct.
- As an **owner**, I want to record adjustments so that legitimate corrections are
  captured transparently rather than hidden.
- As an **owner**, I want expected-vs-actual variance per shift so that I can spot
  shortfalls.

### Non-goals
- No automated bank-feed reconciliation; cash counting is manual entry.

---

## 6. Withdrawals

Backend module: `withdrawal`.

### Scope
- A withdrawal records: amount, destination bank, account number, account holder,
  status, optional reference, requester, and (once claimed) `processedBy`/`claimedAt`/
  `processedAt`/`rejectedReason`.
- Represents moving a tenant's self-order QRIS earnings — sitting in Elkasir's shared
  gateway wallet — out to the tenant's own bank account. Money physically moves outside
  the system (a superadmin does a manual bank transfer); the app only tracks status.
- **Status lifecycle is a two-step claim → complete flow**, not a single action, because
  more than one superadmin can exist and there is no database lock over an outside bank
  transfer:
  ```
  pending --[any superadmin: Claim]--> processing --[ONLY the claimant: Mark Success]--> success
     |                                     |
     +----------[any superadmin: Reject]---+--------------------------------------------> failed
  ```
- **Balance formulas** (do not conflate): `AvailableBalance` (shown everywhere) = self-order QRIS
  revenue − that tenant's `success` withdrawals. The narrower **claimable** check (used to
  authorize `Create`/`Claim`) additionally subtracts that tenant's currently-`processing`
  withdrawals, so a second request/claim can't double-spend money another claim already
  earmarked. A `pending` (unclaimed) request never reserves balance.
- A tenant submitting a request sends a best-effort, generic (no amount/bank details) email to
  every active superadmin.
- Claiming (and therefore completing) a withdrawal is blocked while the tenant is suspended;
  rejecting is not (it's the only way to clear a stale request off a suspended tenant's queue).

### User stories
- As an **owner**, I want to request a withdrawal to a bank account so that I can move
  self-order QRIS earnings out of Elkasir's shared gateway wallet.
- As an **owner**, I want each withdrawal to record bank, account, holder, and a
  status so that the payout is traceable.
- As a **superadmin**, I want to claim a pending withdrawal so that other superadmins know I'm
  the one about to transfer the money, preventing a double-payment race.
- As the **superadmin who claimed a request**, I want to mark it successful once I've actually
  sent the transfer, and no one else should be able to do this for my claim.
- As a **superadmin**, I want to reject a request with a reason, without needing to have claimed
  it, so a stale or invalid request can be cleared even for a now-suspended tenant.
- As a **superadmin**, I want every withdrawal action attributed to who did it and when, so the
  payout trail is fully auditable.

### Non-goals
- No automated bank transfer execution; the transfer itself always happens outside the system
  (a superadmin's own banking app) — this module tracks status/attribution, not the transfer.
- No "release/un-claim" action — if a claimant can't finish, the fix is Reject; the tenant
  resubmits.
- No reserved-balance mechanism for merely-`pending` (unclaimed) requests.

---

## 7. Reports & Analytics

Backend module: `report`.

### Scope
- **Dashboard report**: transaction count, revenue, cash total, QRIS total, recent
  transactions.
- **Sales by day**, **top products**, **category sales**, **payment distribution**
  (cash vs. QRIS counts and totals), **staff performance** (transactions and revenue
  per staff member).

### User stories
- As an **owner**, I want a dashboard summary of today's revenue and transaction
  count so that I can gauge performance at a glance.
- As an **owner**, I want daily sales trends so that I can see how the business moves
  over time.
- As an **owner**, I want top products and category breakdowns so that I know what
  sells.
- As an **owner**, I want the cash-vs-QRIS payment mix so that I understand my cash
  handling exposure.
- As a **manager**, I want per-staff performance so that I can recognize and coach
  the team.

### Non-goals
- No custom report builder or scheduled report exports in this scope.
- No predictive/forecasting analytics.

---

## 8. Payments

Backend module: `payment` — a **DB-configured, multi-app-aware** QRIS/VA gateway. Exactly one
provider (Tripay or Midtrans) is active at a time for `selforder`, configured from **Konsol
Platform**, not `.env` (env vars only bootstrap the DB config once, on first boot, then are never
read again). Alongside it, a SEPARATE, always-on wallet — ElProof, a standalone external product —
is used ONLY for subscription billing (PLAN.md §11); `payment` itself owns no business ledger for
either — only webhook idempotency and a thin order-ref→app-id dispatch index.

### Scope
- Generate a QRIS (or, since Part 2, VA) payment for an order (returns a QR image URL / VA
  account number from the active provider for the customer).
- Track payment status transitions (`pending` → `paid` / `expired` / `failed`).
- Support a `simulated` path for development without live charges.
- Cash remains a first-class payment method, settled manually by the cashier.
- **Registry-driven webhook dispatch**: an incoming gateway callback (Tripay/Midtrans OR ElProof)
  is routed to the correct internal consumer (`selforder` or `subscription`) by looking up its
  `order_ref` in the app registry — no more string-prefix sniffing.
- **Superadmin-configurable**: a superadmin edits the active Tripay/Midtrans provider + credentials
  AND the separate ElProof credentials from Konsol Platform (Konfigurasi Pembayaran).
- **Subscription billing via ElProof** (PLAN.md §11): `subscription`'s charges go through ElProof
  (client-credentials auth, `Elkasir-Billing` app) instead of Elkasir's own wallet — a reconciler
  polls ElProof's status-check endpoint as a fallback, since its webhook relay is best-effort,
  single-attempt. Elkasir no longer exposes its OWN external payment API to other apps (Part 3,
  PLAN.md §10, was removed — see §11).

### User stories
- As a **customer**, I want to pay by scanning a QRIS code so that I can settle from
  my mobile banking/e-wallet app.
- As the **system**, I want to confirm payment via the QRIS gateway so that a self-order is
  marked paid and converted to a transaction.
- As an **owner**, I want unpaid/expired self-orders to be distinguishable from paid
  ones so that I do not fulfil unpaid orders by mistake.
- As a **cashier**, I want cash to be accepted as a payment method so that
  walk-in/counter customers without QRIS can still pay.
- As a **superadmin**, I want to rotate a compromised gateway credential or switch
  sandbox→production from Konsol Platform, without editing server files or restarting.
- As a **superadmin**, I want to register a new external app and see its generated secret
  exactly once, so I can hand it to a partner integrator securely.
- As an **external integrator**, I want to create a charge and be notified when it's paid,
  without needing an Elkasir store of my own.

### Non-goals
- Only QRIS + VA are integrated (no cards, no e-wallet/retail-outlet rails directly).
- No partial refunds, chargeback handling, or payout/disbursement automation via the provider.
- No per-app gateway credentials — still exactly one active provider globally ("one wallet").
- Running two gateways simultaneously.

---

## 9. Platform Console (Konsol Platform) — superadmin surface

Backend module: `platform` (+ contracts-only `platformuser`). The **only** module whose normal
operation is deliberately cross-tenant.

### Scope
- **Tenant lifecycle**: create a new tenant (store + slug + first owner account, one atomic
  operation — the only way to onboard a tenant, no self-registration), list all tenants,
  suspend/reactivate. Suspension blocks that tenant's login and every subsequent request — web
  admin *and* mobile POS — immediately, even on an already-issued token.
- **Revenue reconciliation**: subscription revenue + tenants' undisbursed self-order balances,
  meant to sanity-check against the real gateway balance (not an automated bank reconciliation).
  Per-tenant balance breakdown.
- **Plan catalog management**: create/edit subscription plans (price, period, active/hidden,
  renewal-only lock).
- **Withdrawal processing**: the claim → complete queue and full history (see §6).
- **Superadmin user management**: create, activate/deactivate (never hard-delete, no
  self-deactivation), reset password for other superadmin accounts.
- **Payment gateway config + app registry**: see §8.

### User stories
- As a **superadmin**, I want to onboard a new tenant (store + first owner account) in one step.
- As a **superadmin**, I want to suspend a tenant that's stopped paying/misbehaving, and know
  the lockout is immediate across web and mobile.
- As a **superadmin**, I want a reconciliation view that separates "Elkasir's own subscription
  income" from "tenants' money sitting in the shared wallet, not yet paid out" — these must never
  be conflated into one combined "GMV" figure.
- As a **superadmin**, I want to manage other superadmin accounts, without being able to
  deactivate my own (a safety guard against locking everyone out).

### Non-goals
- No role tiers within `platform_users` (§3).
- No self-service tenant signup — onboarding is always a superadmin action.
- No automatic reconciliation against the real gateway balance — the dashboard's figures are a
  manual sanity-check aid, not an automated audit.

---

## 10. Subscription Billing (tenant → Elkasir)

Backend module: `subscription` — a **separate business domain** from `selforder` (here the store
is the *payer* and Elkasir is the *payee*), even though it reuses the exact same QRIS gateway.
Never shares a row or table with `selforder`'s own payment ledger.

### Scope
- A tenant checks out a plan from its own **Langganan** page; checkout is owner-role-only.
- Plans have a price, a period (days), an active/hidden flag, and a renewal-only lock (§ below).
- Paying extends the subscription period — from the current period's end if still in the future
  (an early renewal stacks on top of remaining time), or from now otherwise.
- **Access gating, computed live on every request, no caching**: a store with no active package
  (no row, wrong status, or a lapsed `current_period_end`) is restricted. POS staff are fully
  blocked. Web admin/owner/manager/viewer are restricted to an allow-list of
  subscription/settings/logout/session routes, with everything else rejected `402 Payment
  Required` (a distinct status from `403`, so the frontend can redirect appropriately instead of
  logging the user out).
- **Renewal-only plans**: a plan can be locked so a subscriber may only ever renew it — never
  switch to nor from any other plan (enforced server-side, not just UI-hidden). Used for
  pre-existing tenants grandfathered in when this billing model was introduced, onto a real,
  named, priced plan rather than a free indefinite one.

### User stories
- As a tenant **owner**, I want to see my current plan, its remaining time, and a manual "check
  payment status" action so I know whether I'm covered without needing to poll.
- As a tenant **owner**, I want to check out a pricier plan (upgrade) when one exists, but not be
  offered a downgrade or a full pricing-comparison grid.
- As a tenant **owner** on a locked (renewal-only) plan, I want a "Perpanjang" (renew) action, and
  I should never see other plans offered to me at all.
- As **POS staff**, if my store's package lapses, I want a clear message telling me to have the
  owner renew — I can't fix billing myself.
- As a **superadmin**, I want existing tenants to be grandfathered with a real grace period when
  this billing model is introduced, not silently locked out on day one.

### Non-goals
- No free trial for brand-new tenants.
- No downgrade flow or full pricing-comparison grid.
- No real-time (SSE) status push or auto-polling — manual "check status" only.
- No invoice-history UI beyond what's already exposed.

---

## Cross-cutting requirements

- **Multi-tenancy**: all queries are scoped to the authenticated user's `store_id` (not
  applicable to `platform`/superadmin routes, the one deliberate exception).
- **Auth**: JWT with access + refresh tokens (except `ActorApp`, which never gets a refresh
  token); **four actor types** — `admin` (web owner/manager), `staff` (POS), `platform`
  (superadmin), `app` (external payment API caller) — roles gate what each actor can do.
- **Access gates enforced live, per request, no caching**: tenant suspension and subscription
  status (§9/§10) are computed fresh on every authenticated request, not just at login.
- **Atomicity**: flows that touch multiple modules (e.g. recording a sale, completing
  a self-order) run under Unit-of-Work so partial writes never persist.
- **API contract**: shapes match the OpenAPI contract; responses use the standard
  envelopes; IDs are ULIDs.

## Global non-goals

- Not a full ERP, accounting suite, or inventory/warehouse system.
- Not an HR/payroll system (staff records are for login and attribution only).
- No offline-first guarantees defined here for the web surfaces (the cashier POS is a
  separate app with its own requirements).
- No marketing/CRM, loyalty points, or promotions engine in this scope.
