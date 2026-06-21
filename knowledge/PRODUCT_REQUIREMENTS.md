# Elkasir — Product Requirements

This document captures the features and user stories of Elkasir, grouped by
functional area. Roles referenced:

- **Owner / Manager** — uses the web admin dashboard (`apps/web`).
- **Cashier (Staff)** — uses the separate Flutter POS app `elkasir_pos`.
- **Customer** — uses the public self-order pages (no account).

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
- No cross-store/global super-admin in this scope; users belong to one store.

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
  status, optional reference, and requester.
- Represents moving funds out of the store's balance to a bank account.

### User stories
- As an **owner**, I want to request a withdrawal to a bank account so that I can move
  earnings out of the store float.
- As an **owner**, I want each withdrawal to record bank, account, holder, and a
  status so that the payout is traceable.
- As an **owner**, I want withdrawals reflected in shift totals so that the drawer
  reconciles after a payout.

### Non-goals
- No automated bank transfer execution in this scope; withdrawal is a recorded
  intent/payout with status, not a live banking integration.

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

Backend module: `payment` (integrates **Xendit** for **QRIS**).

### Scope
- Generate a QRIS payment for an order (returns a QR string for the customer).
- Track payment status transitions (`pending` → `paid` / `expired` / `failed`).
- Support a `simulated` path for development without live charges.
- Cash remains a first-class payment method, settled manually by the cashier.

### User stories
- As a **customer**, I want to pay by scanning a QRIS code so that I can settle from
  my mobile banking/e-wallet app.
- As the **system**, I want to confirm payment via Xendit so that a self-order is
  marked paid and converted to a transaction.
- As an **owner**, I want unpaid/expired self-orders to be distinguishable from paid
  ones so that I do not fulfil unpaid orders by mistake.
- As a **cashier**, I want cash to be accepted as a payment method so that
  walk-in/counter customers without QRIS can still pay.

### Non-goals
- Only QRIS is integrated for digital payment (no cards, no other e-wallet rails
  directly).
- No partial refunds or chargeback handling via the provider in this scope.

---

## Cross-cutting requirements

- **Multi-tenancy**: all queries are scoped to the authenticated user's `store_id`.
- **Auth**: JWT with access + refresh tokens; two actor types (admin web user, POS
  staff); roles gate what each actor can do.
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
