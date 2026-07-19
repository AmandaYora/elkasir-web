# Module Map

Each backend module lives under `apps/api/internal/modules/<module>/` and exposes **only** its
`contracts/` package to other modules. The table below lists responsibility, owned tables, the
public contract a module provides, and the contracts it consumes.

| Module | Responsibility | Owns tables | Provides (contract) | Consumes |
|---|---|---|---|---|
| `auth` | Login (admin + staff + platform superadmin), refresh, logout, `me`; JWT issue/verify; HTTP middleware + principal; tenant-suspension (§2.13) + subscription-gate (§2.15) enforcement on every authenticated request | refresh tokens / sessions; reads `platform_users` directly for login (same pattern as admin_users/staff — see below); reads `stores.status` directly (shared-kernel exception, same class as `settings`/`platform`) | auth middleware + principal (actor, role, storeId) | `subscriptionclient` (read-only `Current`, wired via a post-construction setter — see "auth ↔ subscription wiring" below, NOT a constructor param) |
| `adminuser` | Admin/web users CRUD + password reset (roles: owner/admin/manager/viewer) | `admin_users` | — | auth (mw) |
| `staff` | POS staff CRUD + password reset (roles: cashier/supervisor) | `staff` | — | auth (mw) |
| `product` | Product catalog CRUD, stock adjust/decrement | `products` | `productclient` — `GetForSale`, `ListActive`, `Decrease` (tx-aware) | auth (mw) |
| `category` | Product categories CRUD | `product_categories` | — | auth (mw) |
| `table` | Dining tables CRUD + QR table codes | `tables` | `tableclient` — table lookup by code | auth (mw) |
| `transaction` | Cashier sales: create transaction atomically (service+PPN via settings), list/detail | `transactions`, `transaction_items` | `salesclient` — record sales / read sales aggregates | `productclient`, `shiftclient`, `settingsclient` (orchestration via UoW); auth (mw) |
| `shift` | Cashier shifts: open/close, cash totals, expected vs actual, variance | `shifts` | `shiftclient` — current open shift, accrue sales/cash | auth (mw) |
| `cashmovement` | Cash movements (capital/expense/adjustment) tied to a shift | `cash_movements` | — | `shiftclient`; auth (mw) |
| `withdrawal` | Tenant cash withdrawal requests (bank/account/holder) + the superadmin claim→complete disbursement flow (pending→processing→success/failed) + balance reconciliation; best-effort email ping to all active superadmins on submit | `withdrawals` | `withdrawalclient` — balance reconciliation (`AvailableBalance`/`AvailableBalanceByTenant`/`TotalSuccessfulWithdrawals`) + claim/complete (`ListActive`/`ListAll`/`Claim`/`MarkSuccess`/`MarkRejected`) | `salesclient` (QRIS self-order revenue basis), `platformuserclient` (notify recipients), `internal/platform/mail` (best-effort send); auth (mw) |
| `report` | Dashboard + analytics: sales by day, top products, sales by category, payment distribution, staff performance | (reads aggregates over its own report queries) | — | auth (mw) |
| `payment` | TWO wallets, appID-selected (§11): the ONE active Tripay/Midtrans wallet (§9.1.1, selectable, used by `selforder`) PLUS a SEPARATE, always-on ElProof wallet (used ONLY by `subscription`, appID `AppSubscribe`) — create/quote-fee/verify/parse charges across both + registry-driven webhook dispatch (§9). Part 3's external-facing payment API (Elkasir as a gateway-as-a-service provider for OTHER apps) was REMOVED (§11) — Elkasir is now a client of ElProof instead. Owns NO business ledger — only webhook idempotency (keyed by explicit `provider`, incl. the isolated `"elproof"` namespace), a thin order-ref→app_id dispatch index, DB-backed gateway config (encrypted, now including ElProof's app_id/secret/base_url), and an internal-only app registry (`ELKASIR-SELFORDER`/`ELKASIR-SUBSCRIBE`, seeded by migration, no admin CRUD). Its presentation layer owns `POST /webhooks/payment` (Tripay/Midtrans → in-process consumer) and `POST /webhooks/payment/elproof` (ElProof's relay → the SAME dispatcher, for `subscription`) — superadmin config routes (Tripay/Midtrans + ElProof credentials) live in `platform`, §9.1.10 | `webhook_events`, `payment_charge_apps`, `payment_gateway_config`, `payment_clients` | `paymentclient` — create charge (channel-aware, appID-routed), quote fee, check status (appID-routed), verify/parse webhook (Tripay/Midtrans + separately for ElProof), gateway config CRUD | `auth` (mw); consumed by selforder/subscription (charges) and platform (config passthrough); `subscription` additionally polls `CheckStatus` via its own reconciler (§11 Part C) as a fallback for ElProof's best-effort webhook relay |
| `settings` | Store config: control thresholds, feature flags, **pajak (PPN) & biaya layanan**, **profil toko (nama/telepon/alamat/logo)** (Pengaturan menu) | `settings`, + profile columns on `stores` (shared-kernel exception) | `settingsclient` — read store settings | auth (mw) |
| `selforder` | Customer self-order: public menu, place order, **quote breakdown**, status, claim-code redeem + checkout (orchestrator) | `self_orders`, `self_order_items`, `payments` | — | `productclient`, `salesclient`, `shiftclient`, `tableclient`, `paymentclient`, `settingsclient` (UoW); auth (mw, admin side) |
| `subscription` | Tenant (store) billing to the platform — a SEPARATE business domain from selforder (store is the payer, elkasir is the payee). Plans, subscription period, invoice checkout via the same QRIS gateway | `subscription_plans`, `store_subscriptions`, `subscription_invoices` | `subscriptionclient` — platform revenue aggregate + plan-catalog CRUD (cross-tenant, superadmin-only) | `paymentclient` (create charge, quote fee, verify/parse webhook); auth (mw) |
| `platform` | Superadmin (ActorPlatform) surface: tenant lifecycle (create/list/suspend), cross-tenant reconciliation dashboard + per-tenant balances, plan-catalog management, withdrawal claim/complete processing (enriched with tenant/claimant names), superadmin user management, payment gateway config + app registry (§9.1.10, Part 2). The ONE module whose normal operation is deliberately cross-tenant | none of its own — reads/writes tenant-lifecycle columns (`slug`, `status`) on shared-kernel `stores` (2nd shared-kernel exception, after `settings`' profile columns) | — | `subscriptionclient`, `salesclient`, `withdrawalclient`, `platformuserclient`, `paymentclient` (5 contracts — revenue, plans, withdrawal processing, superadmin accounts, payment config+registry; all read/orchestrate-only); `bootstrap.ProvisionTenant` (tenant creation); auth (mw, ActorPlatform) |
| `platformuser` | Superadmin (`platform_users`) account management — create, activate/deactivate (never hard-delete, no self-deactivation), reset password. Contracts-only: **no HTTP handler, no routes of its own** — `platform` owns `/platform/users/*` and reaches this module only via its contract (same pattern as `payment`) | `platform_users` (CRUD; `auth` separately keeps its own narrow login-lookup queries against the same table, same split already used for `staff`/`admin_users`) | `platformuserclient` — `List`/`Create`/`SetStatus`/`ResetPassword` | — |

> Table ownership is authoritative in [DATABASE_GUIDE.md](DATABASE_GUIDE.md) (derived from migrations). The
> list above is the module-level summary; if they diverge, the migrations + DATABASE_GUIDE win.

## Contract ownership rule

A contract is owned by the **provider** (the capability module), not the consumer:

```txt
product/contracts/      → productclient.Client       (consumed by transaction, selforder)
shift/contracts/        → shiftclient.Client         (consumed by transaction, cashmovement, selforder)
transaction/contracts/  → salesclient.Client         (consumed by selforder, platform, withdrawal)
table/contracts/        → tableclient.Client         (consumed by selforder)
payment/contracts/      → paymentclient.Client       (consumed by selforder, subscription, platform)
subscription/contracts/ → subscriptionclient.Client  (consumed by platform, auth)
withdrawal/contracts/   → withdrawalclient.Client    (consumed by platform)
platformuser/contracts/ → platformuserclient.Client  (consumed by platform, withdrawal)
```

### `auth` ↔ `subscription` wiring (Phase B1.5) — a deliberate exception to constructor injection

Every other consumer above receives its dependency as a plain constructor parameter. `auth`
cannot: `subscription.New(...)` itself needs `auth`'s `Middleware` to protect its own routes, so
a `subscriptionclient.Client` constructor param on `auth.New` would be a circular dependency in
`app.go`. Instead, `auth.Module` exposes `SetSubscriptionClient(c)`, called once in `app.go`
immediately after `subscription.New(...)` returns — module construction order (`auth` first,
`subscription` later) stays unchanged. This is the one place in the codebase where a contract is
wired post-construction rather than through `New(...)`; see `apps/api/internal/app/app.go` and
`auth/infrastructure/subscription_gate.go`.

Correct: `selforder.application` → `productclient.Client.GetForSale(...)`
Forbidden: `selforder` → `product.infrastructure.Repo` or a SQL join `self_order_items JOIN products`.

`payment` now has TWO consumers (`selforder`, `subscription`) — the whole reason its `CreateCharge`
records no ledger row of its own: each consumer owns and writes its own payment ledger (selforder's
`payments`, subscription's `subscription_invoices`), so the two business domains' money never share
a table. See `paymentclient.Charge.Provider`.

## Orchestrators (cross-module, atomic via Unit-of-Work)

- **`transaction`** composes a cashier sale: `productclient.Decrease` + `salesclient` record +
  `shiftclient` accrue, all in one DB transaction.
- **`selforder`** composes a customer self-order checkout across product/sales/shift/table/payment.

No orchestrator calls another module's repository directly — only its contract client, which runs on
the shared UoW transaction when one is open. `subscription` is NOT an orchestrator (it never touches
another module's table) — it only consumes `paymentclient.Client` like any other consumer.

## One shared payment webhook, registry-driven dispatch (PLAN.md §9, Part 2)

Tripay/Midtrans each support only ONE callback URL per merchant account, yet `selforder`,
`subscription`, and (eventually) other registered apps all need webhook delivery through the
same gateway — still exactly ONE wallet (`payment_gateway_config`, one active provider). `payment`
now owns this dispatch itself (`payment/presentation`, its first-ever HTTP route,
`POST /webhooks/payment`) instead of the composition root: it verifies + parses the callback via
its own gateway, checks `webhook_events` idempotency, then looks up which registered `app_id`
created the charge (`payment_clients` + the thin `payment_charge_apps` order-ref index, written
by `CreateCharge`/`CreateChannelCharge`) and dispatches to whichever Go consumer is registered
for that `app_id`. Every `CreateCharge` call now takes an `appID` (`paymentclient.AppSelfOrder` /
`paymentclient.AppSubscribe`) instead of the old `"sub_"` ref-prefix convention that used to live
in `subscription/domain` — that convention (and the composition-root's hardcoded two-consumer
branch) no longer exists. Internal consumers (`selforder`, `subscription`) are registered once in
`app.go` via `paymentMod.RegisterConsumer(appID, consumer)`, right after both modules are
constructed — this is still the one place allowed to know concrete consumers exist; neither
consumer module imports the other or knows about the registry.

**Since PLAN.md §11**, `subscription`'s charges are created through a SEPARATE wallet (ElProof)
instead of the shared Tripay/Midtrans one `selforder` uses — but the SAME dispatch mechanism above
is reused unchanged: a second inbound route, `POST /webhooks/payment/elproof`, normalizes ElProof's
own relay format into the same `paymentclient.WebhookEvent` shape and calls the same `Dispatch`,
which still resolves `AppSubscribe` from `payment_charge_apps` and calls the same registered
consumer. `subscription` itself required NO changes — `CreateCharge`/`ApplyWebhookEvent` are
called exactly as before; the gateway selection by `appID` is entirely internal to `payment`.

## `platform` — the one deliberately cross-tenant module

Every module above scopes every query by `store_id` from the authenticated principal — this is
the whole multi-tenancy contract. `platform` is the SINGLE exception: it's the superadmin
(`ActorPlatform`) surface, and by definition needs to read/manage across ALL tenants (tenant
list, reconciliation dashboard, plan catalog, withdrawal processing, superadmin accounts). It
never breaks other modules' isolation to do this — it reaches cross-tenant data ONLY through
each contract's deliberately-unscoped methods: `subscriptionclient.PlatformRevenue`/
`ListAllPlans`/`CreatePlan`/`UpdatePlan`, `salesclient.PlatformSelfOrderQrisRevenue`/
`PlatformSelfOrderQrisRevenueByTenant`, `withdrawalclient.AvailableBalanceByTenant`/
`TotalSuccessfulWithdrawals`/`ListActive`/`ListAll`/`Claim`/`MarkSuccess`/`MarkRejected`, and
`platformuserclient.List`/`Create`/`SetStatus`/`ResetPassword` (documented inline at each).
Tenant/claimant **names** shown alongside a withdrawal (`platform/application.WithdrawalView`)
are joined in Go across these contracts' results — never a SQL join across modules' tables.
Tenant CREATION goes through `bootstrap.ProvisionTenant` (store + default settings + owner admin
account, one transaction) — the same "infra-level provisioning" pattern `bootstrap.Seed` already
used, not a new precedent.

## Multi-tenancy audit fixes (migration `000016_platform`)

Two entry points resolved a tenant from a value that was only unique **per-store**, without any
tenant identifier in the request — found during a multi-tenancy audit, before `platform` existed
to onboard more than one tenant:

- **Self-order QR (`FindTableByCode`)** — `dining_tables.code` is unique only per `(store_id,
  code)`. Fixed by requiring a **tenant slug** in the URL (`/order/<slug>/<tableCode>`,
  `tableclient.Client.FindByCode(ctx, storeSlug, code)`, resolved via a join to shared-kernel
  `stores.slug`). Every tenant gets a slug at creation time (`platform.CreateTenant`).
- **Staff POS login (`GetStaffByUsername`)** — `staff.username` was unique only per `(store_id,
  username)`, but the login endpoint is hit by the SAME mobile APK across all tenants with no
  tenant context at all (`{username, password}` only) — adding one would be a breaking change to
  `elkasir_mobile`. Fixed instead by making `staff.username` **globally unique** (mirrors
  `admin_users.username`, migration 000006) — zero change to the mobile app's request contract.
