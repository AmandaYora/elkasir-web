# Module Map

Each backend module lives under `apps/api/internal/modules/<module>/` and exposes **only** its
`contracts/` package to other modules. The table below lists responsibility, owned tables, the
public contract a module provides, and the contracts it consumes.

| Module | Responsibility | Owns tables | Provides (contract) | Consumes |
|---|---|---|---|---|
| `auth` | Login (admin + staff), refresh, logout, `me`; JWT issue/verify; HTTP middleware + principal | refresh tokens / sessions | auth middleware + principal (actor, role, storeId) | — |
| `adminuser` | Admin/web users CRUD + password reset (roles: owner/admin/manager/viewer) | `admin_users` | — | auth (mw) |
| `staff` | POS staff CRUD + password reset (roles: cashier/supervisor) | `staff` | — | auth (mw) |
| `product` | Product catalog CRUD, stock adjust/decrement | `products` | `productclient` — `GetForSale`, `ListActive`, `Decrease` (tx-aware) | auth (mw) |
| `category` | Product categories CRUD | `product_categories` | — | auth (mw) |
| `table` | Dining tables CRUD + QR table codes | `tables` | `tableclient` — table lookup by code | auth (mw) |
| `transaction` | Cashier sales: create transaction atomically (service+PPN via settings), list/detail | `transactions`, `transaction_items` | `salesclient` — record sales / read sales aggregates | `productclient`, `shiftclient`, `settingsclient` (orchestration via UoW); auth (mw) |
| `shift` | Cashier shifts: open/close, cash totals, expected vs actual, variance | `shifts` | `shiftclient` — current open shift, accrue sales/cash | auth (mw) |
| `cashmovement` | Cash movements (capital/expense/adjustment) tied to a shift | `cash_movements` | — | `shiftclient`; auth (mw) |
| `withdrawal` | Cash withdrawal requests (bank/account/holder) | `withdrawals` | — | auth (mw) |
| `report` | Dashboard + analytics: sales by day, top products, sales by category, payment distribution, staff performance | (reads aggregates over its own report queries) | — | auth (mw) |
| `payment` | QRIS gateway (Tripay/Midtrans, selectable): create/quote-fee/verify payment | (payment refs) | `paymentclient` — create charge, **quote fee**, status | auth (mw) |
| `settings` | Store config: control thresholds, feature flags, **pajak (PPN) & biaya layanan**, **profil toko (nama/telepon/alamat/logo)** (Pengaturan menu) | `settings`, + profile columns on `stores` (shared-kernel exception) | `settingsclient` — read store settings | auth (mw) |
| `selforder` | Customer self-order: public menu, place order, **quote breakdown**, status, claim-code redeem + checkout (orchestrator) | `self_orders`, `self_order_items` | — | `productclient`, `salesclient`, `shiftclient`, `tableclient`, `paymentclient`, `settingsclient` (UoW); auth (mw, admin side) |

> Table ownership is authoritative in [DATABASE_GUIDE.md](DATABASE_GUIDE.md) (derived from migrations). The
> list above is the module-level summary; if they diverge, the migrations + DATABASE_GUIDE win.

## Contract ownership rule

A contract is owned by the **provider** (the capability module), not the consumer:

```txt
product/contracts/    → productclient.Client      (consumed by transaction, selforder)
shift/contracts/      → shiftclient.Client         (consumed by transaction, cashmovement, selforder)
transaction/contracts/→ salesclient.Client         (consumed by selforder)
table/contracts/      → tableclient.Client         (consumed by selforder)
payment/contracts/    → paymentclient.Client       (consumed by selforder)
```

Correct: `selforder.application` → `productclient.Client.GetForSale(...)`
Forbidden: `selforder` → `product.infrastructure.Repo` or a SQL join `self_order_items JOIN products`.

## Orchestrators (cross-module, atomic via Unit-of-Work)

- **`transaction`** composes a cashier sale: `productclient.Decrease` + `salesclient` record +
  `shiftclient` accrue, all in one DB transaction.
- **`selforder`** composes a customer self-order checkout across product/sales/shift/table/payment.

No orchestrator calls another module's repository directly — only its contract client, which runs on
the shared UoW transaction when one is open.
