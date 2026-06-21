# Elkasir — Product Requirements Document (PRD)

> Multi-tenant Point-of-Sale (POS) for Food & Beverage businesses.
> This document describes the **product**: what Elkasir is, who it serves, the
> problems it solves, its scope, feature modules, key business flows, and how we
> measure success. It is intentionally product-level — implementation details
> live in [`SYSTEM_DESIGN.md`](./SYSTEM_DESIGN.md) and [`DEPLOYMENT.md`](./DEPLOYMENT.md).

---

## 1. Purpose

Elkasir is a cloud-deliverable POS platform built specifically for **F&B
businesses** — cafés, warungs, kiosks, and small-to-mid restaurants. It gives an
owner a single system to:

- manage their menu (catalog) and dining tables,
- let customers **self-order** by scanning a QR code at their table,
- accept payment by **QRIS** or **cash**,
- run cashier sales and shift-based cash handling through a companion cashier app,
- and watch the business through a **web admin dashboard** with sales reports.

The product is **multi-tenant**: many independent stores share one deployment,
and every record is isolated by `store_id`. A store never sees another store's
data.

Elkasir is delivered as a single Go service that simultaneously serves:

1. a **React 19 web admin dashboard** for owners and managers, and
2. **customer self-order pages** (QR table → menu → order → pay).

A separate **Flutter app (`elkasir_pos`, not in this repo)** is the cashier POS
used by staff at the counter; it consumes the same API.

---

## 2. Problem statement

Small F&B operators are stuck between two bad options:

- **Generic/cash-register POS** that does not understand F&B workflows (tables,
  self-order, shifts, cash drawer reconciliation) and offers no real reporting.
- **Enterprise POS suites** that are expensive, heavy to set up, and assume a
  single large business rather than many small independent ones.

Specific pains Elkasir targets:

- **Order-taking is slow and error-prone.** Staff manually relay table orders;
  queues form at peak hours.
- **Cash handling is opaque.** Owners cannot tell whether the cash in the drawer
  matches what was sold during a shift, or who took money out.
- **No trustworthy numbers.** Owners lack daily revenue, top-product, and
  payment-mix visibility to make decisions.
- **Payment friction.** Accepting QRIS digitally usually means juggling a
  separate device or app.

---

## 3. Target users & personas

| Persona | Where they work | What they need from Elkasir |
|---|---|---|
| **Owner** | Web admin dashboard | Full control: catalog, staff/admin users, reports, withdrawals, sees all money flows. The top tenant authority for a store. |
| **Manager / Admin** | Web admin dashboard | Day-to-day operations: manage products/categories/tables, approve shift closings, review incoming self-orders, read reports. |
| **Viewer** | Web admin dashboard | Read-only access to dashboards and reports (e.g. an accountant or investor). |
| **Cashier (Staff)** | Flutter cashier app (`elkasir_pos`) | Open/close shifts, ring up sales, take cash/QRIS, manage the drawer. |
| **Supervisor (Staff)** | Flutter cashier app | Cashier duties plus elevated actions such as approving cash movements and shift closing. |
| **Customer (guest)** | Self-order web pages (phone browser) | Scan the table QR, browse the menu, place an order, and pay — no login. |

Admin web roles: `owner`, `admin`, `manager`, `viewer`.
Staff (POS) roles: `cashier`, `supervisor`.

---

## 4. Scope

### 4.1 In scope

- Multi-tenant store isolation (every entity scoped by `store_id`).
- Product catalog with categories.
- Dining tables with QR-code-driven customer self-order.
- Self-order checkout with **QRIS** (via payment provider) and **cash** options.
- Cashier sales captured by the Flutter POS app via the API.
- Shift lifecycle: open, track cash/QRIS sales, and close with cash reconciliation.
- Cash movements (capital injection, expenses, adjustments) tied to a shift.
- Withdrawals (transfer of takings to a bank account) with status tracking.
- Reports & analytics: dashboard summary, daily sales, top products, category
  sales, payment distribution, staff performance.
- Two staff management surfaces: **POS staff** (cashiers/supervisors) and
  **admin/web users** (owner/admin/manager/viewer).
- Authentication via JWT for two principal types: **admin** and **staff**.

### 4.2 Out of scope (for this product)

- The cashier hardware/app itself — `elkasir_pos` is a **separate Flutter
  project** that only consumes this API.
- Kitchen display systems, printers, and hardware peripheral drivers.
- Inventory procurement / supplier management / purchase orders (only simple
  product stock is tracked).
- Accounting/ledger integrations, payroll, loyalty programs, and online delivery
  marketplace integrations.
- A database-as-a-container deployment — MySQL is expected at host/OS level.

---

## 5. Feature modules

Elkasir is organized as a set of feature modules. Each is a business capability;
the boundaries are mirrored in the backend architecture (see
[`SYSTEM_DESIGN.md`](./SYSTEM_DESIGN.md)).

### 5.1 Catalog (products & categories)
- CRUD of **products** (`name`, `sku`, `price`, `cost`, `stock`, `status`,
  optional `imageUrl`, optional `categoryId`).
- CRUD of **categories** (`name`, `sortOrder`), each reporting its `productCount`.
- Products power both the admin views and the customer-facing menu.

### 5.2 Tables & self-order
- CRUD of **dining tables** (`code`, `name`, `area`, `seats`, `status`).
- A **public menu** endpoint resolves a table code into the table info plus the
  store's active categories and products.
- **Self-order**: a customer places an order against a table; the order has a
  lifecycle `placed → preparing → completed`, a `paymentMethod` (`qris`/`cash`),
  and a `paymentStatus` (`pending → paid` / `unpaid` / `expired` / `failed`).
- Admins see self-orders in an **Incoming Orders** ("Pesanan Masuk") view.

### 5.3 Staff / admin management
- **POS staff** (`cashier`, `supervisor`) with `username` login, used by the
  Flutter POS.
- **Admin/web users** (`owner`, `admin`, `manager`, `viewer`) with `email` login,
  used by the web dashboard. Tracks `lastActiveAt`.

### 5.4 Sales (transactions)
- A **transaction** is a completed sale. Source is either `cashier` (rung up in
  the POS app) or `self_order` (placed by a customer).
- Holds line items, `subtotal`, `discount`, `tax`, `total`, `paymentMethod`,
  `amountReceived`, `changeAmount`, and links to the originating `shiftId`,
  `tableId`, `selfOrderId`, and `cashierId` by **primitive ID**.

### 5.5 Shifts & cash
- A **shift** is a cashier's working session with an `initialCash` float.
- During the shift Elkasir accumulates `cashSales`, `qrisSales`,
  `additionalCapital`, `expenses`, `withdrawals`, `adjustments`, and
  `drawerOpenCount`.
- On close it computes `expectedCash`, records `actualCash`, derives the
  `variance`, and records who approved the close (`closeApprovedBy`).
- **Cash movements** record drawer changes during a shift: `capital`, `expense`,
  or `adjustment`, optionally requiring an approver.

### 5.6 Withdrawals
- Records a payout of takings to a bank account (`amount`, `bank`, `account`,
  `holder`) with a `status` and optional `reference`, plus who requested it.

### 5.7 Reports
- **Dashboard report**: transaction count, revenue, cash total, QRIS total, plus
  a list of recent transactions.
- **Sales by day**, **top products**, **category sales**, **payment
  distribution** (cash vs QRIS counts/totals), and **staff performance**.

### 5.8 Payments
- Integrates a payment provider (Xendit, sandbox-capable) for **QRIS**.
- Generates a QR string for the customer to scan, then confirms payment via
  provider callback/webhook; **cash** is settled directly.
- If the QRIS provider is not configured, the QRIS path is disabled (cash only),
  and self-order payment may run in a simulated mode for testing.

---

## 6. Key business flows

### 6.1 Self-order checkout (customer)

```
Customer scans QR on table
        │
        ▼
Open public menu (table code → table + categories + products)
        │
        ▼
Add items → place order  ──►  Self-order created (status=placed, payment=pending)
        │
        ├─ paymentMethod = qris → provider returns QR string → customer pays
        │                          → webhook marks paymentStatus = paid
        │
        └─ paymentMethod = cash → settled at counter by staff
        │
        ▼
Checkout: paid self-order is converted into a Transaction (source=self_order)
        │
        ▼
Order progresses placed → preparing → completed
```

The conversion of a **paid self-order into a transaction** is an atomic,
cross-module operation (self-order + transaction + shift totals must all succeed
or all roll back).

### 6.2 Cashier sale (staff via Flutter POS)

```
Cashier (with an OPEN shift) builds a cart in elkasir_pos
        │
        ▼
Take payment: cash (compute change from amountReceived) or QRIS
        │
        ▼
Create Transaction (source=cashier) — items, totals, paymentMethod
        │
        ▼
Atomically: persist transaction + decrement product stock + roll the sale
            into the open shift's cash/QRIS totals
```

### 6.3 Shift open / close with cash reconciliation

```
Open shift:  cashier records initialCash → shift.status = open
        │
        ▼
During shift: cashSales, qrisSales, capital, expenses, adjustments,
              withdrawals, and drawerOpenCount accumulate
        │
        ▼
Close shift: system computes expectedCash
             = initialCash + cashSales + additionalCapital
               − expenses − withdrawals ± adjustments
             cashier counts the drawer → actualCash
             variance = actualCash − expectedCash
        │
        ▼
A supervisor/manager approves the close (closeApprovedBy); shift.status = closed
```

Reconciliation surfaces over/short drawers so owners can investigate variances.

---

## 7. Success criteria

**Product outcomes**

- A customer can go from scanning a table QR to a paid order without staff
  intervention for QRIS.
- A store can run a full day: open shift → take cashier + self-orders →
  close shift with a reconciled drawer.
- Owners get accurate daily revenue, payment mix, and top-product numbers without
  manual tallying.
- Multiple stores operate on one deployment with strict data isolation.

**Quality bars**

- **Tenant isolation:** no store can read or mutate another store's data; the
  `store_id` is always derived from the authenticated principal, never the
  request body.
- **Cash integrity:** shift `expectedCash` and `variance` are always derivable
  from recorded movements; transactions and shift totals never drift apart
  (guaranteed by atomic, all-or-nothing checkout/sale flows).
- **Availability:** the service exposes liveness/readiness probes (`/healthz`,
  `/readyz`) and runs as a single self-contained container.
- **Payment correctness:** a self-order is only converted to a completed
  transaction once payment is confirmed (paid), and QRIS confirmation is driven
  by the provider webhook.
