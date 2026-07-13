# PLAN — Platform Console + Tenant Subscription + Withdrawal Reconciliation + Payment Gateway Multi-App Abstraction + External Payment API

**Status:** **Part 1** (§0–§8: Platform Console + Tenant Subscription + Withdrawal Reconciliation) is COMPLETE. All backend phases (B0–B6) and all frontend phases (F0–F6) implemented and verified — `go build && go vet && go test` clean, `tsc --noEmit` clean, `eslint .` clean (0 new errors), and every flow walked end-to-end through the real UI (see §8 for the full verification log and the real bugs found + fixed along the way).
**Part 2** (§9: Payment Gateway Multi-App Abstraction) is **COMPLETE** — planned 2026-07-12, implemented and verified the same day. Evolved the `payment` module (formerly a single-gateway, no-consumer-identity wrapper) into a DB-configured, multi-app-aware gateway that can eventually serve other SaaS products, while keeping exactly one Tripay/Midtrans merchant account ("one wallet"). All backend phases (`PB0`–`PB4`) and frontend phases (`PF0`–`PF2`) implemented — `go build && go vet && go test` clean, `tsc --noEmit` clean, `eslint .` clean, and every flow (config edit, app registry CRUD, self-order + subscription charge/webhook dispatch) verified through real HTTP calls and a real browser. §9.2/§9.3's verification notes carry the full log; §9.7 marked the external-facing API as deliberately NOT built yet — that is what Part 3 designs.
**Part 3** (§10: External Payment API) is **COMPLETE** — designed 2026-07-12, implemented and verified the same day. Built the external-facing HTTP surface §9.7 had deferred: a registered `kind=external` app (§9.1.3) can now call in over the network — `POST /auth/app/token` (client-credentials, no refresh token) to authenticate as a new `ActorApp`, then `POST/GET /external/payments/*` to create charges, poll status, and list channels — and receives signed webhook relays when its charges get paid, reusing Part 2's registry rather than inventing a parallel one. All backend phases (`EB0`–`EB5`) implemented — `go build && go vet && go test` clean, no frontend phases needed (§10.1.7). Every flow (token issuance incl. no-refresh-token, charge creation, idempotency 409, HMAC-signed webhook relay verified byte-for-byte against the app's own secret, cross-app 404 isolation, live deactivation revocation, both rate limits) verified through real HTTP calls against the running dev API — see EB5's verification note in §10.2 for the full log. Written integrator-facing reference at `docs/EXTERNAL_PAYMENT_API.md`.
**Branch:** `feat/tenant-subscription-billing` (uncommitted changes present — backend work from this feature; do not discard).
**Ordering rule (explicit, from the user):** finish and green-light **all** backend phases of a part before starting that part's frontend phases. For Part 1: §3 (B0–B6) fully green before §4 (F0–F6). For Part 2: §9.2 (`PB0`–`PB4`) fully green before §9.3 (`PF0`–`PF2`). Part 3 has no frontend phases (§10.1.7). Do not interleave within a part.
**Purpose of this file:** checkpoint/handoff. Read top to bottom, resume from the first unticked box in the current phase. Decisions marked "LOCKED" in §2, §9.1, and §10.1 were debated at length across long design conversations — don't re-open them without a new, explicit reason from the user.
**Env access:** the user has explicitly authorized editing `apps/api/.env` directly (it's gitignored — real secrets belong only there, never in this file or `.env.example`). Real SMTP credentials for Phase B4 are already staged there — see the note inside Phase B4 below. Part 2 changes this policy for payment-gateway credentials specifically — see §9.1.2. Part 3 introduces no new `.env` values.

---

## 0. Context

Elkasir is a multi-tenant POS SaaS (`elkasir_web`). This feature adds a superadmin/platform layer ("Konsol Platform") and a tenant-facing subscription page. It grew to include tenant cash-withdrawal processing because self-order QRIS and subscription QRIS both settle into **one shared Tripay/Midtrans merchant account** — only the superadmin can actually disburse a tenant's earnings. Because more than one superadmin account can exist, withdrawal processing is a **two-step claim → complete** flow (not a single button) to prevent two superadmins from both manually transferring money for the same request before either records it. Tenants get a best-effort email ping (to all active superadmins) when they submit a request. A suspended tenant loses all access — web and mobile — immediately, and a superadmin cannot claim/pay out a suspended tenant's pending withdrawal. A small set of superadmin-management + audit-history screens round out Konsol Platform.

Full rationale for every decision below lives in this conversation's history — this file carries only the *decisions*.

---

## 1. Backend (core) — DONE, verified, do not redo

Verified via `go build ./...`, `go vet ./...`, `go test ./...` (all green).

- Migration `000015_subscription_billing` — `subscription_plans`, `store_subscriptions`, `subscription_invoices`.
- Migration `000016_platform` — `platform_users` table; `stores.slug`/`stores.status`; `staff.username` now globally unique; `refresh_tokens.actor` gains `'platform'`.
- Module `subscription` — tenant billing, QRIS checkout, `subscriptionclient.Client` (platform revenue aggregate + plan CRUD).
- Module `platform` — tenant lifecycle (create/list/suspend), revenue dashboard, plan catalog management. It already consumes `subscriptionclient` + `salesclient`; §3 adds two more contracts (`withdrawalclient`, `platformuserclient` — Phase B5) and many more routes it owns.
- `auth` extended: third actor `ActorPlatform`, `POST /auth/platform/login`, reads `platform_users` directly for login (same pattern as admin_users/staff — CRUD ownership is separate, see Phase B3).
- `table`/`selforder`: self-order QR requires a tenant slug. Frontend for this fix is done.
- `docs/`, `knowledge/` updated to match.

### API surface already available (unchanged by this plan)
```
POST /auth/platform/login   { email, password } → { accessToken, refreshToken, expiresIn, user }
GET  /platform/tenants  |  POST /platform/tenants  |  PATCH /platform/tenants/{id}/status
GET  /platform/plans    |  POST /platform/plans    |  PATCH /platform/plans/{id}
GET  /subscription  |  GET /subscription/plans  |  POST /subscription/checkout  |  GET /subscription/invoices
GET  /settings   (now includes storeSlug)
```
`GET /platform/revenue`'s response shape changes in **Phase B5** — don't build frontend against its current shape.

---

## 1a. Pre-flight code audit (2026-07-11) — corrections folded into §3/§4

Before starting B0, every technical assumption in §1/§3/§4 was re-verified against the live code (not just re-running `go build ./... && go vet ./... && go test ./...` and `tsc --noEmit`, which are all still clean). A handful of assumptions the phases below were originally written against turned out to be slightly off. None of this reopens a §2 LOCKED decision — every fix is mechanical and is folded inline into the relevant phase's checklist below, not tracked separately. Index, so a resuming agent isn't surprised a checklist looks denser than its intro paragraph implies:

- **B1.5** — `subscriptionclient.Client`'s public interface doesn't expose `Current(ctx, storeID)` yet (only the concrete `subscription/application.Service` does) — B1.5 now starts by extending the contract. Also, wiring `auth → subscriptionclient` creates a real constructor-order cycle in `app.go` (`subscription.New` itself needs `auth`'s `Middleware` to protect its own routes) — B1.5 now specifies late/setter injection to break that cycle, not a module reorder.
- **B1** — this is the first time `auth`'s middleware performs any per-request I/O at all (today it's pure JWT parse/verify, zero DB or contract calls). Not a design change — §2.13's "no caching" decision stands as-is — just worth knowing going in, since there's no existing per-request-lookup pattern in this file to copy.
- **F0** — `setOnUnauthorized` in `http-client.ts` is currently a single global callback slot, not domain-keyed, and `RequestConfig` is currently unexported. Both are now explicit F0 checklist items, not implicit.
- **F2** — `AppSidebar`/`AppHeader`/`AppLayout` currently import `useAuthStore` directly inside their bodies, not via props. "Accept props with a tenant-store default" isn't enough to make them reusable for Konsol Platform — the internal import itself has to come out.
- **F4** — `http-client.ts` today only has special interceptor handling for `401` (single-flight refresh + retry-once); there is no existing `403` branch. The `402` branch F4 adds is new logic, not a variation of an existing 401/403 pair.
- **F5** — the withdrawal `status` vocabulary is already 3-way inconsistent *before* this feature touches it: the real DB enum is `pending/processing/success/failed` (unchanged by B0), but `openapi.yaml` currently documents `[pending, paid, rejected]` and `WithdrawalStatusBadge` currently handles 8 different keys (looks copy-pasted from the self-order payment badge). F5 now reconciles all three, not just adds a 4th status to the badge.

---

## 2. LOCKED design decisions

1. **Two fully separate identity domains.** Separate Zustand stores, separate token storage (`elkasir_access_token` pair vs `elkasir_platform_access_token` pair), separate session-user shape. One shared Axios instance; every request explicitly declares `tokenDomain`. Logging into one must never authenticate the other.

2. **Login pages differ; everything after login reuses the same components.** Same split-screen shell, different copy/preview. Dashboard shell, sidebar, header, stat cards, tables, modals — reused via props, not duplicated.

3. **Konsol Platform's visual identity = tenant admin dashboard's visual identity.** Only distinguishing signal: sidebar subtitle "Konsol Platform" vs "Admin POS". No dark theme, no new palette.

4. **Langganan (tenant subscription page) is deliberately minimal:** current plan + countdown + upgrade CTA (only if pricier plan exists) + manual "Cek status pembayaran" button (**no polling, no SSE**). `status==="none"` shows a plan-picker. Checkout/upgrade is owner-only in the UI.

5. **Konsol Platform's revenue view is a reconciliation tool, not a sales report.** Cash self-order payments never touch the gateway — never count them toward anything withdrawal-related.
   ```
   Pendapatan Langganan  +  Saldo Tenant Belum Dicairkan  =  Total Termonitor
   ```
   should equal the real Tripay/Midtrans balance (manual sanity check, not automated). The old combined-methods "GMV Self-Order" metric is retired entirely from Konsol Platform.

6. **Two different balance formulas, for two different purposes — do not conflate them:**
   - **`AvailableBalance(tenant)`** (displayed everywhere — Ringkasan, Revenue Tenant, tenant's own page; reconciliation-accurate) = `SelfOrderQrisRevenue(tenant) − Σ(that tenant's status='success' withdrawals)`. Money hasn't left the gateway until a withdrawal is `success`, so this is the number that should sum against the real gateway balance.
   - **Claimable check** (internal validation only, used by both `withdrawal.Create` and the `Claim` action, never shown as its own UI figure) = `AvailableBalance(tenant) − Σ(that tenant's currently status='processing' withdrawals)`. This is narrower — money that's been *claimed* by a superadmin (about to be transferred) is still physically sitting in the gateway, but it's "spoken for," so a second request/claim must not also count it as free.
   - **No "reserve balance for pending (unclaimed) requests" mechanism** — a `pending` request doesn't reduce either figure above. Multiple pending requests can look valid against the same balance; whichever gets *claimed* first locks in that amount via the claimable check, the rest fail at claim-time with a clear error.

7. **Withdrawal processing is a two-step claim → complete flow, not a single action — because more than one superadmin account can exist and the actual money transfer happens outside the system (in a banking app), where there's no database-level lock to protect against double-payment.**
   ```
   pending --[any superadmin: Klaim]--> processing --[ONLY the claimant: Tandai Sukses]--> success
      |                                     |
      +----------[any superadmin: Tolak]----+----------------------------------------------> failed
   ```
   - `Claim` (pending→processing): any active superadmin. Records `processed_by` + `claimed_at`. Runs the claimable check (§2.6) **and** the tenant-suspension check (§2.14).
   - `Tandai Sukses` (processing→success): **only** the superadmin who claimed it (`processed_by` must match the acting principal) — asserting "I personally completed this transfer." No further balance check needed (already gated at claim time). Records `processed_at`.
   - `Tolak` (pending→failed **or** processing→failed): any active superadmin, with a required reason. Doesn't move money, so no ownership restriction and **not** gated by tenant-suspension status. Records `processed_by` (whoever rejected) + `processed_at` + `rejected_reason`.
   - Both the claim step and the final step use an atomic conditional UPDATE (`WHERE status = <expected>`, check rows-affected) — the same idiom already used elsewhere in this codebase (`VoidTransaction`, `MarkSubscriptionInvoicePaid`) — so a genuine concurrent double-click is *also* impossible at the database level, on top of the UI-visible "claimed by X" signal that prevents the human-coordination problem this whole flow exists for.
   - **No "release/un-claim" action** — if a claimant can't finish, the fix is `Tolak`; the tenant resubmits. Considered, deliberately not built (§5).

8. **Every withdrawal action is attributable.** `processed_by` is set at claim time and carried through to the final outcome; it's populated on rejection too, not just success. `platform_users` accounts are **never hard-deleted** (deactivate only) — deleting the actor would orphan this trail.

9. **Konsol Platform user management (`platformuser` module) is in scope.** No role tiers within platform users. Self-deactivation is blocked (a superadmin cannot deactivate their own account). Never hard-delete.

10. **Tenant submitting a withdrawal triggers a best-effort email to every active platform user — generic content, no sensitive data.** The email says only "a new withdrawal request is waiting for review, open Konsol Platform" (+ a link built from the existing `PublicBaseURL` config) — **no tenant name, no amount, no bank details.** Reasoning: (a) `withdrawal` module has no clean contract to resolve a tenant's display name (it only holds `store_id`, and no existing contract exposes `stores.name` to it — adding one just for an email subject was rejected as unnecessary), and (b) financial details sitting in an email inbox is an avoidable information-disclosure surface. Sending is fire-and-forget in a background goroutine — a failed/misconfigured SMTP send must **never** block, slow down, or fail the tenant's `POST /withdrawals` request, and must never be visible to the tenant in any way (no field in any tenant-facing response reveals an email was attempted). If `SMTP_HOST` isn't configured, the mail sender no-ops (logs once) — same pattern already used for optional object storage.
    - No email on claim, success, or rejection — only on initial tenant submission.
    - No SLA/deadline enforcement — "diproses dalam 2 hari kerja" (see §2.11) is informational copy only, not a tracked deadline with automated escalation.

11. **Tenant-facing copy addition (Withdrawals page):** a static info line "Proses pencairan memakan waktu hingga 2 hari kerja" near the request form/list. Purely informational.

12. **Sidebar grouping** (mirrors the tenant sidebar's own grouped convention):
    ```
    Ikhtisar   → Ringkasan
    Tenant     → Tenant, Revenue Tenant
    Keuangan   → Penarikan, Riwayat Penarikan
    Sistem     → Paket, User Platform
    ```

13. **A suspended tenant loses access entirely — immediately, not "eventually once their token expires."** Login (`LoginAdmin`, `LoginStaff`), token refresh, *and every subsequent authenticated request* (via the `Authenticate` middleware) must reject with 403 if the principal's `store_id` resolves to `stores.status = 'suspended'`. This is a straight per-request DB check — no caching layer, no eventual-consistency window; the app's scale doesn't warrant the complexity of caching a value that must be immediately authoritative. Because both `elkasir_web` and `elkasir_mobile` (POS staff) authenticate through this same `auth` module and the same middleware, enforcing this once in the Go backend covers both clients — no client-specific work needed. `ActorPlatform` principals (no `store_id`) are exempt from this check entirely. The rejection message is distinct from the existing "akun ini nonaktif" (inactive *account*) message — this is about the *tenant*, not the individual user, so the remedy text differs ("Toko Anda sedang dinonaktifkan. Hubungi platform." — not "hubungi pemilik toko," since the owner is locked out too).

14. **Withdrawal claiming is separately blocked while the tenant is suspended — but rejecting a request is not.** `Claim` (and therefore `MarkSuccess`, which requires an existing claim) must fail if the withdrawal's tenant is currently suspended — this is a distinct business rule from §2.13 (it's the *superadmin's* action being blocked, not the tenant's own access), so it's enforced separately, inside `withdrawal.Claim` itself (reads `stores.status` directly — a narrow, read-only, precedented kind of shared-kernel access, same justification class as `settings`/`platform`'s existing exceptions, just for a status flag rather than a write). `MarkRejected` is **not** gated by tenant status — rejecting never moves money, and it's the only way to clear a stale request off a suspended tenant's queue.

15. **Tenant `active` status is tied to having exactly one currently-active subscription package** — decided 2026-07-11, resolving the gap previously flagged as an open question (old §6 item 1; kept below in §6 with only the residual sub-decision still open). This is a **second, independent gate**, separate from §2.13's manual suspension — don't conflate the two.
    - **"Punya paket aktif" (has an active package)** = the tenant has a `store_subscriptions` row with `status = 'active'` **AND** `current_period_end >= NOW()`. Nothing else counts: no row at all (`"none"`), `trial`, `past_due`, `canceled`, or `expired` (by literal status or by lapsed period) all fail this check equally. There is no trial/grace concept in this rule — only a paid, currently-in-period plan satisfies it.
    - **Computed live, no caching** — same philosophy as §2.13. `current_period_end` is compared against `NOW()` on every gated login/request; there is no cron flipping `status` to `'expired'`. (Confirmed by reading `subscription/infrastructure/repository.go`: nothing in the current, already-verified subscription code ever transitions a row's `status` to `'expired'` — so a live period-end comparison is required regardless of the stored status literal.)
    - **New tenants are unchanged: `ProvisionTenant` still creates no `store_subscriptions` row.** The consequence is now an intentional rule rather than a gap — a brand-new tenant is "package-inactive" from the moment of creation until its owner completes a subscription checkout. No free ride, not even temporarily.
    - **Not a third `stores.status` enum value.** `stores.status` stays `active|suspended`, owned entirely by §2.13's manual superadmin action. Package-inactive is evaluated *after* the §2.13 suspension check, with weaker (partial, not total) enforcement:
      - `ActorPlatform` — exempt, same as §2.13.
      - `stores.status == 'suspended'` (manual) still wins outright — full lockout, existing message, checked first.
      - Otherwise, if the tenant has no active package:
        - **`ActorStaff`** (POS — cashier/supervisor, `elkasir_mobile`) — **fully blocked**: rejected at `LoginStaff`, at refresh, and at every subsequent request. Message is distinct from both existing suspension messages: *"Toko ini belum memiliki paket langganan aktif. Hubungi pemilik toko untuk memperbarui langganan."* (staff can't fix billing themselves, so — unlike §2.13's message — this one points to the *owner*, not the platform.)
        - **`ActorAdmin`** (web dashboard — owner/admin/manager/viewer) — **partially blocked**: `LoginAdmin` and refresh still succeed. Every subsequent request is allowed through **only** if it hits an explicit allow-list of subscription/account routes (`GET/POST /subscription*`, `GET /settings`, `POST /auth/logout`); anything else is rejected with a **new, distinct status — `402 Payment Required`** (deliberately not `403`, so the frontend can tell "unauthorized" apart from "unpaid" and redirect accordingly), message *"Paket langganan toko Anda tidak aktif. Perbarui paket langganan untuk melanjutkan."* Checkout itself stays owner-only (§2.4, unchanged) — any admin role can still reach Langganan to see this and hand it to the owner.
    - **Explicitly does not touch withdrawal claiming (§2.14).** A tenant's self-order QRIS earnings are the tenant's own money, unrelated to whether they've paid *Elkasir's* subscription fee — `withdrawal.Claim` keeps checking only `stores.status == 'suspended'`, not package status. Don't extend §2.14's check to cover this.
    - **Existing (pre-feature) tenants are NOT grandfathered/exempted.** They get backfilled with a real, active `store_subscriptions` row via a one-time data migration (§3, new Phase B1.5) so this rule doesn't retroactively lock out a tenant for a plan they were never asked to buy. Which specific plan they're backfilled onto is a small decision still open — see Phase B1.5's checklist.

---

## 3. Backend implementation — DO THIS FIRST, IN THIS ORDER

Nothing in §4 (frontend) starts until every box below is ticked and `go build ./... && go vet ./... && go test ./...` is clean. Run that verify command after *each* phase.

**Phase order and why:** `B0` (schema) and `B1` (auth) are independent of everything else and of each other — done first as the foundation. `B1.5` (subscription-gated access, §2.15) depends only on `B1` (same middleware/service files, sequenced right after to avoid rework) and the `subscription` module — independent of `B2`–`B5`, feeds only into `B6`'s final verify. Per §1a, `B1.5` also extends `subscriptionclient.Client`'s interface and wires `auth`'s dependency on it via a setter (not a constructor param) to avoid a construction-order cycle with `subscription.New` needing `auth`'s `Middleware` — see B1.5's checklist. `B2` (withdrawal) needs `B0`'s migration. `B3` (platformuser) is independent. `B4` (email) needs `B3`'s contract. `B5` (platform module) needs both `B2` and `B3`'s contracts, so it comes after both. `B6` closes out with docs + the full-system verify.

```
B0 (migration+transaction) ─┐
B1 (auth suspension)  ──────┼─→ B2 (withdrawal) ─┐
B3 (platformuser)  ─────────┴─→ B4 (email) ───────┼─→ B5 (platform routes) → B6 (docs + verify)
                                                    │
                              B2 ──────────────────┘

B1 (auth suspension) ─→ B1.5 (auth subscription gate, §2.15) ─→ B6 (docs + verify)
```

### Phase B0 — Migration + `transaction` module (self-order QRIS aggregates)

- [x] New migration (next number after `000016`): `ALTER TABLE withdrawals` — add `processed_by CHAR(26) NULL` (primitive ID → `platform_users`, no FK), `claimed_at DATETIME NULL`, `processed_at DATETIME NULL`, `rejected_reason VARCHAR(255) NULL`. `status` enum unchanged (`pending,processing,success,failed`) — all four values are now meaningfully used.
- [x] `db/queries` — replace the combined-methods `SumSelfOrderRevenue` with QRIS-only versions: `SumSelfOrderQrisRevenue :one` (cross-tenant), `SumSelfOrderQrisRevenueByStore :one` (+ `store_id=?`), `SumSelfOrderQrisRevenueGroupedByStore :many` (`GROUP BY store_id`).
- [x] `salesclient.Client`: remove `PlatformSelfOrderRevenue`; add `PlatformSelfOrderQrisRevenue(ctx) (int64, error)`, `SelfOrderQrisRevenueForStore(ctx, storeID string) (int64, error)`, `PlatformSelfOrderQrisRevenueByTenant(ctx) ([]TenantAmount, error)`.
- [x] Regenerate sqlc, `go build ./...` clean.

### Phase B1 — Tenant suspension enforcement (§2.13)

Independent of B0 — touches only `auth`.

- [x] `auth.sql` (or wherever auth's queries live): new query `GetStoreStatus :one` — `SELECT status FROM stores WHERE id = ? LIMIT 1`.
- [x] `auth/infrastructure/middleware.go` — `Middleware` gains access to run that query (a `*sqlcgen.Queries` reference or a small dedicated dependency). After JWT parse succeeds: if `p.Actor` is `ActorAdmin` or `ActorStaff` (i.e. has a non-empty `StoreID`), look up the store's status; if `'suspended'`, respond `httpx.Forbidden("Toko Anda sedang dinonaktifkan. Hubungi platform.")` and stop — don't call `next`. Skip the check entirely for `ActorPlatform`.
- [x] `auth/application/service.go` — add the same check to `LoginAdmin`, `LoginStaff`, and `principalFromRefresh` (both the admin and staff branches) — reject with the same message. This is in addition to (not a replacement for) the existing `admin_users.status`/`staff.status` (individual account) checks already there.
- [x] `go build ./... && go vet ./...` clean.
- [x] Manual test: suspend a test tenant via `PATCH /platform/tenants/{id}/status`. Confirm (a) a fresh login attempt is rejected, (b) an *already-issued, still-unexpired* access token now gets 403 on the very next API call — not just at next login/refresh, (c) a refresh attempt with an existing valid refresh token is rejected, (d) reactivating the tenant immediately restores access.

### Phase B1.5 — Subscription-gated access (§2.15)

Depends on B1 (touches the same middleware/service files — sequenced right after to avoid rework) and the `subscription` module. **Correction from §1a: `subscriptionclient.Client`'s public interface does not yet expose a `Current` method — only the concrete `subscription/application.Service` does — so this phase starts by extending the contract, not consuming an already-done one.** Independent of B2–B5.

- [x] **New, first:** `subscription/contracts/client.go` — add a `Subscription` DTO (`Status string; CurrentPeriodEnd time.Time`, mirroring the fields the concrete `application/service.go`'s `Current` already returns) and add `Current(ctx context.Context, storeID string) (Subscription, error)` to the `subscriptionclient.Client` interface. Adjust `application.Service.Current`'s return type (or add a thin adapter method) so the concrete service still satisfies the extended interface. `subscription.module.go`'s existing `Module.Client subscriptionclient.Client = svc` wiring needs no further change.
- [x] **New, wiring:** `auth` module gains a `subscriptionclient.Client` dependency, but **not as a constructor param** — `subscription.New(...)` itself already requires `auth`'s `Middleware` (to gate its own routes), so a constructor param on `auth.New` would be a circular dependency in `app.go`. Instead, add a setter (e.g. `func (m *auth.Module) SetSubscriptionClient(c subscriptionclient.Client)`, or the same directly on `Middleware`) called once in `app.go`, immediately after `subMod := subscription.New(...)` runs and before any routes are registered. Construction order in `app.go` stays `auth` first, `subscription` later, unchanged.
- [x] `auth` module gains its now-real `subscriptionclient.Client` dependency (via the setter above) — reads `Current(ctx, storeID)` only (narrow, read-only contract access, same justification class as §2.14's `stores.status` read).
- [x] `internal/platform/httpx` — add a new typed error/helper for `402 Payment Required` (e.g. `httpx.PaymentRequired(message string)`), alongside the existing `Forbidden` helper.
- [x] `auth/infrastructure/middleware.go` — after the existing §2.13 suspension check passes (tenant not suspended): compute "has active package" (`subscriptionclient.Current(ctx, storeID)`: `status == "active" && currentPeriodEnd >= now`).
  - For `ActorStaff`: if no active package, reject `403` with the new staff-specific message (§2.15) and stop.
  - For `ActorAdmin`: if no active package, allow the request through **only** if the resolved route matches the allow-list. **Correction found during F4 E2E testing:** chi's `RouteContext().RoutePattern()` collapses any sub-path under a `Route()`-mounted group into a wildcard leaf (e.g. `/api/v1/subscription/plans` reports as `/api/v1/subscription/*`), so exact-pattern matching silently failed for every subscription route except the mount root. Match on `r.URL.Path` with prefix comparison instead (`/api/v1/subscription`, `/api/v1/settings`, etc., matching either exactly or with a trailing `/...`) — this is also more faithful to this section's own original wording ("`GET/POST /subscription*`", a prefix, not a closed list).
  - Skip this whole check for `ActorPlatform`. **Also skip it (fail open) if the subscription client hasn't been set yet** — only relevant during the brief construction window in `app.go` before the setter above runs; by the time the server accepts requests it's always set.
- [x] `auth/application/service.go` — `LoginStaff` and the staff branch of `principalFromRefresh` gain the same package-inactive check (reject `403`, same message as the middleware). `LoginAdmin` and the admin branch of `principalFromRefresh` are **not** gated here — only at the per-route middleware level, so an admin can always log in far enough to reach Langganan.
- [x] Define the allow-list as a small constant (in `auth`, or wherever route patterns are declared) covering: `GET /subscription`, `GET /subscription/plans`, `POST /subscription/checkout`, `GET /subscription/invoices`, `GET /settings`, `POST /auth/logout`, **and `GET /auth/me`** (found during F4 implementation: `/auth/me` sits behind the same `Authenticate` middleware as everything else, so without it the frontend's session-restore call would itself 402 for a package-inactive owner — and the frontend would misread that as an invalid session and log them out, contradicting this section's own stated intent that "an admin can always log in far enough to reach Langganan"). Confirm exact paths against `api-endpoints.ts` before finalizing.
- [x] New data migration (next number in sequence) — one-time backfill for existing tenants that currently have zero `store_subscriptions` rows: insert one `active` row per such `store_id`, `current_period_start = NOW()`. **Open decision, confirm before writing this migration:** which `plan_id` to backfill onto — a dedicated new hidden plan (`is_active = 0`, so it never shows in the tenant-facing picker) seeded specifically for this purpose (recommended — keeps real pricing plans clean of a synthetic "legacy" entry), or an existing real plan? And how long a `current_period_end` — recommended: far enough out (e.g. years, not days) that no legacy tenant gets silently relocked by this very feature before anyone revisits the decision.
- [x] `go build ./... && go vet ./...` clean.
- [x] Manual test: (a) a brand-new tenant (no subscription row) — owner can log in and reaches only Langganan-scoped routes, staff login is rejected; (b) after a real checkout succeeds, both unlock fully; (c) let a real subscription's `current_period_end` lapse (or fake it in the DB) — the same lockout re-triggers without any `status` column ever changing; (d) a legacy/backfilled tenant has full access immediately after the migration runs.

### Phase B2 — `withdrawal` module: contract, claim/complete flow, balance validation

Depends on B0 (new columns).

- [x] `withdrawal` module gains a `salesclient.Client` dependency (constructor param).
- [x] New file `withdrawal/contracts/client.go` — `withdrawalclient.Client`:
  ```go
  AvailableBalance(ctx, storeID string) (int64, error)                       // §2.6, reconciliation basis
  AvailableBalanceByTenant(ctx) ([]TenantBalance, error)                     // same basis, all tenants
  TotalSuccessfulWithdrawals(ctx) (int64, error)                             // cross-tenant, feeds Ringkasan
  ListActive(ctx) ([]Withdrawal, error)                                      // status IN (pending, processing), cross-tenant
  ListAll(ctx, filter ListFilter) ([]Withdrawal, total int64, error)         // any status, paginated, cross-tenant
  Claim(ctx, id string, actorID string) error                                // pending→processing; claimable check §2.6 AND
                                                                              // tenant-suspension check §2.14 (reads stores.status
                                                                              // directly); sets processed_by+claimed_at
  MarkSuccess(ctx, id string, actorID string) error                          // processing→success; requires actorID==processed_by; sets processed_at
  MarkRejected(ctx, id string, actorID string, reason string) error          // pending|processing→failed; any actor; sets processed_by/processed_at/rejected_reason
  ```
  `Claim` and `MarkSuccess` use atomic conditional UPDATEs (`WHERE status = <expected>` [+ `AND processed_by = ?` for `MarkSuccess`], check rows-affected → typed "already claimed/processed" error if 0). `AvailableBalanceByTenant` merges `salesclient.PlatformSelfOrderQrisRevenueByTenant()` with this module's own `GROUP BY store_id` sum of `status='success'` withdrawals in Go (not SQL — different modules' tables).
- [x] `withdrawal.Create` (existing, tenant-facing): reject with `httpx.Unprocessable(...)` if `amount > AvailableBalance(storeID) − Σ(this store's processing withdrawals)` (the claimable-basis check, §2.6).
- [x] `withdrawal.module.go`: expose `Client withdrawalclient.Client` field.
- [x] New tenant-facing endpoint: `GET /withdrawals/balance` → `{ availableBalance: int64 }`.
- [x] `go build ./... && go vet ./...` clean.

### Phase B3 — New `platformuser` module

Independent of B0–B2.

- [x] New module `apps/api/internal/modules/platformuser` — contracts-only-facing, no presentation/routes of its own. `domain/`, `infrastructure/` (repo over `platform_users`), `application/`, `contracts/`.
- [x] `platformuserclient.Client`:
  ```go
  List(ctx) ([]PlatformUser, error)
  Create(ctx, in CreateInput) (PlatformUser, error)             // CreateInput = {Name, Email, Password string}
  SetStatus(ctx, actingUserID, targetID, status string) error   // rejects if actingUserID == targetID && status == "inactive"
  ResetPassword(ctx, id, newPassword string) error
  ```
  No delete method (§2.8/§2.9).
- [x] `platformuser.module.go` — wiring file (every module gets one, even contracts-only ones — `payment.module.go` is the existing precedent for a module with no presentation/routes). Exposes `Module{Client platformuserclient.Client}`.
- [x] `platformuser/domain/events/events.go` — stub file for scaffold consistency with every other module (empty is fine; no domain events are actually emitted by this module).
- [x] `auth` module untouched — still reads `platform_users` directly for login.
- [x] `go build ./... && go vet ./...` clean.

### Phase B4 — Email notification on withdrawal submission

Depends on B3 (`platformuserclient`) and B2 (`withdrawal.Create` exists to hook into).

**Credentials already staged.** `apps/api/.env` (gitignored, not this file) already has real
Hostinger SMTP credentials filled in under `SMTP_HOST`/`SMTP_PORT`/`SMTP_USERNAME`/`SMTP_PASSWORD`/
`SMTP_FROM_EMAIL`/`SMTP_FROM_NAME` — sending account is `cs@elcodelabs.com` on `smtp.hostinger.com`,
port 587 (STARTTLS; Hostinger also exposes 465/implicit-TLS if the chosen SMTP implementation needs
that instead). `apps/api/.env.example` has the same keys with placeholder values only — never put
the real password there or in this file. When implementing `internal/platform/mail`, just read
`config.Config.SMTP.*` — no further credential-gathering step needed, they're already in place.

- [x] New package `internal/platform/mail`:
  ```go
  type Config struct { Host, Port, Username, Password, FromEmail, FromName string }
  func (c Config) Enabled() bool { return c.Host != "" && c.FromEmail != "" }
  type Sender struct{ /* cfg */ }
  func New(cfg Config) *Sender
  func (s *Sender) Send(ctx context.Context, to []string, subject, body string) error  // no-ops (returns nil) if !cfg.Enabled()
  ```
  Plain SMTP (`net/smtp` or a minimal no-dependency SMTP lib) — no third-party email API SDK, consistent with this project's self-hosted, single-container philosophy. Send one email per recipient (not a shared `To`/`Bcc` list) — small recipient count, avoids header-exposure concerns.
- [x] `config.Config` gains an `SMTP` sub-struct; `.env.example` (both root and `apps/api/`) gains:
  ```env
  SMTP_HOST=
  SMTP_PORT=587
  SMTP_USERNAME=
  SMTP_PASSWORD=
  SMTP_FROM_EMAIL=noreply@elkasir.app
  SMTP_FROM_NAME=Elkasir Platform
  ```
- [x] `withdrawal` module gains two more constructor params: `platformuserclient.Client` and `*mail.Sender`.
- [x] `withdrawal.Create`: after the row is successfully inserted, spawn `go s.notifyPlatformUsers(context.WithoutCancel(ctx), ...)` — best-effort, never affects the response to the tenant:
  ```go
  func (s *Service) notifyPlatformUsers(ctx context.Context) {
      users, err := s.platformUsers.List(ctx)
      if err != nil { slog.Warn("withdrawal: notify platform users", "err", err); return }
      link := s.publicBaseURL + "/platform/withdrawals"
      subject := "Permintaan pencairan baru menunggu ditinjau"
      body := fmt.Sprintf("Ada permintaan pencairan baru yang menunggu ditinjau. Buka Konsol Platform untuk meninjau: %s", link)
      for _, u := range users {
          if u.Status != "active" { continue }
          if err := s.mailer.Send(ctx, []string{u.Email}, subject, body); err != nil {
              slog.Warn("withdrawal: send notification failed", "to", u.Email, "err", err)
          }
      }
  }
  ```
  No tenant name, amount, or bank details in the email body (§2.10). `publicBaseURL` reuses the existing config value already used for payment callback URLs — no new env var for the link itself.
- [x] `go build ./... && go vet ./...` clean. Manually verify with a real (or local dev SMTP catcher like MailHog/Mailpit) that the email fires on create and that a deliberately-broken SMTP config doesn't affect the `POST /withdrawals` response.

### Phase B5 — `platform` module: consume the two new contracts, add routes

Depends on B2 (`withdrawalclient`) and B3 (`platformuserclient`).

- [x] Constructor gains `withdrawalclient.Client` and `platformuserclient.Client` params (now 4 contracts total: `subscriptionclient`, `salesclient`, `withdrawalclient`, `platformuserclient`), wired in `app.go`.
- [x] Rewrite `GET /platform/revenue`:
  ```json
  { "subscriptionRevenue": 0, "tenantAvailableBalance": 0, "totalMonitored": 0 }
  ```
  `tenantAvailableBalance = salesclient.PlatformSelfOrderQrisRevenue() − withdrawalclient.TotalSuccessfulWithdrawals()`. Drop the old `selfOrderRevenue` field.
- [x] New `GET /platform/tenants/revenue` → joins `platform`'s own tenant list (`stores` — name/slug) with `withdrawalclient.AvailableBalanceByTenant()` by `storeID`, in Go. Sorted by balance descending.
- [x] New withdrawal-processing routes:
  ```
  GET   /platform/withdrawals                → withdrawalclient.ListActive(), each row enriched with
                                                 tenant name + (if processing) claimant's name
  PATCH /platform/withdrawals/{id}/claim      → withdrawalclient.Claim(id, principal.SubjectID)
  PATCH /platform/withdrawals/{id}/success    → withdrawalclient.MarkSuccess(id, principal.SubjectID)
  PATCH /platform/withdrawals/{id}/reject     body {reason} → withdrawalclient.MarkRejected(id, principal.SubjectID, reason)
  GET   /platform/withdrawals/history         → withdrawalclient.ListAll(filter), rows enriched with
                                                 tenant name + processor name (via platformuserclient.List() → ID→name map)
  ```
- [x] New platform-user-management routes:
  ```
  GET   /platform/users
  POST  /platform/users                        body {name, email, password}
  PATCH /platform/users/{id}/status             body {status}
  PATCH /platform/users/{id}/reset-password     body {password}
  ```
- [x] All routes in this phase gated `RequireActor(ActorPlatform)`.
- [x] `go build ./... && go vet ./...` clean.

### Phase B6 — Docs + final backend verification

Depends on everything above.

- [x] Update `knowledge/MODULE_MAP.md`, `knowledge/DATABASE_GUIDE.md`, `docs/DB_SCHEMA.md`, `apps/api/db/README.md` to match everything above (new module, new contracts, new columns, new endpoints).
- [x] `packages/api-contract/openapi.yaml` — **deliberately NOT updated for the new `/platform/withdrawals/*`, `/platform/tenants/revenue`, `/platform/users/*`, `/withdrawals/balance` endpoints in this phase.** Same precedent as the earlier subscription/platform work: no frontend consumer exists yet, so a speculative OpenAPI entry would be unverifiable. Revisit and fill in once Phase F3/F5's frontend actually consumes them — don't let this silently stay stale forever, but don't block B6 on it either.
- [x] `go build ./... && go vet ./... && go test ./...` — all clean. This is the gate before §4 starts.
- [x] Manual smoke test (curl/Postman, no frontend exists yet): create a self-order QRIS payment for a test tenant → confirm balance → create a withdrawal within limit (succeeds, confirm notification email fires if SMTP configured) → try one exceeding the limit (422) → **claim** the valid one as superadmin A → confirm it's now `processing` with `processed_by=A` → try to **claim it again** as superadmin B (should fail — already processing) → try `success` as superadmin B (should fail — not the claimant) → mark `success` as superadmin A (should succeed, balance drops) → confirm `/platform/withdrawals/history` shows the full trail with correct names/timestamps.
- [x] Manual smoke test for §2.13/§2.14: suspend the test tenant → confirm its admin/staff login and an already-issued token are both rejected (per B1's test) → confirm a *new* withdrawal claim attempt for that tenant is rejected (§2.14) while a **reject** action on one of its pending requests still succeeds → reactivate the tenant → confirm login and claiming both work again immediately.

---

## 4. Frontend implementation — only after §3 (B0–B6) is fully green

**Phase order:** `F0` (token plumbing) is the foundation for everything else. `F1` (platform login) needs `F0`. `F2` (shell) needs `F1`. `F3` (platform pages) needs `F2` and the backend endpoints from `B5`/`B6`. `F4` (Langganan) and `F5` (tenant Withdrawals updates) only need `F0` and can be done independently of `F1`–`F3` if you're splitting work — both are tenant-side, not platform-side. `F6` closes out with full-system manual verification.

### Phase F0 — Shared plumbing (token domain isolation)

- [x] `http-client.ts`: `TokenDomain = "tenant" | "platform"`; domain-keyed token storage; `tokenStore.*(..., domain = "tenant")` defaults preserve every existing call site; **export** `RequestConfig` (currently declared but not exported) and add `RequestConfig.tokenDomain?`; request/response interceptors and refresh logic become domain-aware; `setOnUnauthorized` — **currently a single module-level callback variable, not domain-keyed** — becomes domain-aware (e.g. `Record<TokenDomain, (() => void) | null>`, or two named setters) so a platform-session callback can't silently overwrite the tenant session's, or vice versa.
- [x] `api-endpoints.ts`:
  ```ts
  auth: { ...existing, platformLogin: "/auth/platform/login" },
  platform: {
    tenants: "/platform/tenants", tenantsRevenue: "/platform/tenants/revenue", revenue: "/platform/revenue",
    plans: "/platform/plans", withdrawals: "/platform/withdrawals", withdrawalHistory: "/platform/withdrawals/history",
    users: "/platform/users",
  },
  subscription: { root: "/subscription", plans: "/subscription/plans", checkout: "/subscription/checkout" },
  withdrawalBalance: "/withdrawals/balance",
  ```
- [x] Verify: `tsc --noEmit` clean, tenant login unaffected.

### Phase F1 — Platform auth store + login page

- [x] `platform.types.ts` — mirror every DTO from §3 (camelCase).
- [x] `platform-auth.store.ts` — `usePlatformAuthStore`, twin of `useAuthStore`, `tokenDomain:"platform"` throughout, `restore()` rejects if `actor !== "platform"`.
- [x] `AuthLayout.tsx` refactor — extract `BrandPanelShell({headline, description, preview, tagline})`; `AuthLayout` takes `brandPanel: ReactNode` + guard config. Existing `/login` must render pixel-identical after.
- [x] `PlatformLoginPage.tsx` — near-copy of `LoginPage.tsx`, "Masuk ke Konsol Platform".
- [x] `route-paths.ts` + `public.routes.tsx`.
- [x] Verify: login with seeded superadmin works; `/login` unaffected.

### Phase F2 — Reusable dashboard shell, grouped nav

Per §1a: `AppSidebar`/`AppHeader`/`AppLayout` currently import `useAuthStore` directly inside their bodies — an optional prop that *defaults* to calling that store wouldn't actually decouple them, it'd just hide the coupling. Below, each of these three explicitly loses its internal `useAuthStore` import in favor of props supplied by the layout that wraps it.

- [x] `AppSidebar.tsx` — accept `groups` (nav sections) as a required prop instead of reading a hardcoded module-level array; `subtitle?` prop for "Konsol Platform" vs "Admin POS". The tenant call site (inside `AppLayout.tsx`) passes today's hardcoded groups as its prop value, so tenant behavior is unchanged.
- [x] `AppHeader.tsx` — **remove its direct `useAuthStore` import**; accept `user`/`onLogout`/role-label as props instead. Update its one call site (inside `AppLayout.tsx`) to read from `useAuthStore` there and pass the values down.
- [x] `AppLayout.tsx` — **remove its direct `useAuthStore` import** too (today its session guard reads `status`/`user` inline); extract the guard into something the layout takes as props or a small wrapper (e.g. a `store`/`loginPath` param) so `PlatformLayout` can supply `usePlatformAuthStore` without duplicating the whole layout body.
- [x] `platformNav.ts` — grouped per §2.12: Ikhtisar(Ringkasan) / Tenant(Tenant, Revenue Tenant) / Keuangan(Penarikan, Riwayat Penarikan) / Sistem(Paket, User Platform).
- [x] `PlatformLayout.tsx` — twin of the now-decoupled `AppLayout.tsx`, guards on `usePlatformAuthStore`, supplies `platformNav`'s groups and the "Konsol Platform" subtitle.
- [x] Verify: `tsc --noEmit` clean; tenant `/dashboard` unaffected (same nav, same header, same guard behavior as before F2).

### Phase F3 — Platform pages (7 pages)

- [x] `platform.service.ts` — every method from §3, all passing `{tokenDomain:"platform"}`.
- [x] `schemas/{tenant,plan,platform-user}.schema.ts`.
- [x] **`PlatformOverviewPage.tsx`** ("Ringkasan") — 3 `StatCard`s (Pendapatan Langganan / Saldo Tenant Belum Dicairkan / Total Termonitor with reconciliation caption).
- [x] **`PlatformTenantsPage.tsx`** — Table + create Modal + suspend/activate.
- [x] **`PlatformTenantRevenuePage.tsx`** ("Revenue Tenant") — Table: Tenant | Saldo Belum Dicairkan, descending. Read-only.
- [x] **`PlatformWithdrawalsPage.tsx`** ("Penarikan") — active requests (`pending`+`processing`): Tenant | Jumlah | Bank info | Saldo tersedia | Status. Per §2.7:
  - `pending` row: "Klaim" button (any superadmin).
  - `processing` row: "Diklaim oleh [Nama]" badge + "Tandai Sukses" button **enabled only if the logged-in superadmin is the claimant** (else disabled/hidden with a tooltip) + "Tolak" button (always enabled, any superadmin, opens a reason input).
  - Surface backend rejection errors clearly (e.g. claim race lost, non-claimant trying to mark success, or the tenant-suspended case from §2.14).
- [x] **`PlatformWithdrawalHistoryPage.tsx`** ("Riwayat Penarikan") — paginated, any status: Tenant | Jumlah | Status badge | Diajukan | Diklaim (tanggal+nama) | Diselesaikan (tanggal) | Alasan (if rejected).
- [x] **`PlatformPlansPage.tsx`** — Table + create/edit Modal.
- [x] **`PlatformUsersPage.tsx`** ("User Platform") — Table (Nama, Email, Status, Dibuat) + create Modal + Reset Password + Aktifkan/Nonaktifkan (self-row's toggle disabled client-side too, mirroring the backend guard).
- [x] Wire all 7 into routing (`route-paths.ts`, new `platform.routes.tsx` wrapped in `PlatformLayout`, added to `app/routes/index.tsx`).
- [x] Verify manually: full tenant/plan CRUD; full withdrawal lifecycle including the two-superadmin race scenario from B6's manual test, now through the real UI (need ≥2 platform user accounts to test properly — create a second one via User Platform first).

### Phase F4 — Tenant-facing "Langganan" page + app-wide subscription-lock guard (§2.15)

- [x] Types/services: `getCurrent`, `listPlans`, `checkout` (default tenant token domain).
- [x] `SubscriptionPage.tsx` per §2.4. Gate checkout/upgrade button to owner role.
- [x] Routing + sidebar entry ("Sistem", after "Pengaturan").
- [x] `http-client.ts` (tenant domain only — platform-domain requests are never gated by this): add a **new** response-interceptor branch for `402` (per §1a: today only `401` has special handling — single-flight refresh + retry-once — there is no existing `403` branch to mirror, so this is new logic, not a variation of an existing pair). On `402`, redirect any non-Langganan route to the Langganan page and surface a persistent "Paket tidak aktif" state instead of the normal error toast.
- [x] `AppLayout.tsx`/tenant route guard — when locked, render a reduced shell (header only, nav either hidden or all-disabled except Langganan) so the owner isn't stuck on a page they can no longer use, and isn't shown a raw 402 error.
- [x] Verify manually: a package-inactive tenant's owner can reach and use only Langganan; every other tenant route redirects/locks with the new copy; a staff account for the same tenant can't log in at all (§2.15).

### Phase F5 — Tenant Withdrawals page: balance + status labels + info copy

Depends on B2's `GET /withdrawals/balance`.

- [x] `withdrawals.service.ts` — add `getBalance()` → `GET /withdrawals/balance`.
- [x] `WithdrawalsPage.tsx`:
  - Show "Saldo dapat dicairkan: Rp X" near the top; client-side validate the create-form amount against it (server authoritative per B2).
  - Add the static info line from §2.11: "Proses pencairan memakan waktu hingga 2 hari kerja."
  - Per §1a, the withdrawal `status` vocabulary is 3-way inconsistent today, independent of this feature — reconcile all three before touching the badge:
    1. `packages/api-contract/openapi.yaml`'s `Withdrawal.status` enum is currently `[pending, paid, rejected]` — stale, doesn't match the real DB enum (`pending|processing|success|failed`, unchanged by B0). Fix it to the real 4 values as part of this phase (a correction to the *existing* `/withdrawals` endpoints' docs, separate from B6's deliberate deferral of documenting the *new* B2/B5 endpoints).
    2. `withdrawal.types.ts`'s `Withdrawal.status` is currently a bare `string` — change it to a literal union of the 4 real values so this can't silently drift again.
    3. `WithdrawalStatusBadge.tsx` currently maps **8** keys (`success, completed, pending, unpaid, processing, failed, cancelled, expired` — looks copy-pasted from the self-order payment badge) — replace with exactly: `pending`="Menunggu"(warning), `processing`="Sedang diproses"(info/primary), `success`="Berhasil"(success), `failed`="Ditolak"(danger, show `rejectedReason` if present). Don't just add a 4th case to the existing 8.
- [x] Verify: over-limit submission rejected client- and server-side; a `processing` request displays correctly while a superadmin has it claimed.

### Phase F6 — Final verification pass

- [x] `tsc --noEmit` clean. `eslint .` — no new errors.
- [x] Manual cross-context isolation checks (§2.1): platform-only, tenant-only, both-at-once.
- [x] Full withdrawal lifecycle walkthrough through the real UI, with 2 superadmin accounts, reproducing the claim-race scenario end to end.
- [x] Confirm the tenant never sees any trace of the email notification (no field, no toast, no console log visible to them).
- [x] Repeat §2.13/§2.14's suspension scenario through the real UI: suspend a test tenant from the Tenant page, confirm its admin is immediately logged out / blocked on next action (not just at their next login), confirm a pending withdrawal for that tenant can't be claimed from the Penarikan page (clear error shown) but can still be rejected, then reactivate and confirm everything works again.
- [x] Update this file's checkboxes, or note explicitly what's left and why.

---

## 5. Explicit non-goals (don't build unless separately asked)

- Invoice history UI for tenant subscriptions.
- Device-pairing for POS staff login (`elkasir_mobile`) — separate project.
- Dark-mode/distinct visual theme for Konsol Platform — rejected, §2.3.
- Real-time (SSE) subscription status push, and auto-polling as a substitute — manual refresh only, §2.4.
- Downgrade flow / full pricing-comparison grid on Langganan.
- Combined-methods (cash+QRIS) "GMV Self-Order" reporting anywhere in Konsol Platform — retired, §2.5.
- A "reserve balance for pending (unclaimed) withdrawal requests" mechanism — §2.6.
- A "release/un-claim" action for withdrawals — §2.7; use Tolak instead.
- Automatic bank-transfer API integration — processing stays manual.
- Automatic reconciliation against the real Tripay/Midtrans balance.
- Role tiers within `platform_users`, hard-delete of platform users.
- Email notifications for anything other than initial tenant submission (no email on claim/success/reject) — §2.10.
- Sensitive data (tenant name, amount, bank details) in the notification email — §2.10.
- Automated SLA/deadline tracking or escalation for the "2 hari kerja" copy — §2.10/§2.11, informational text only.
- Caching the tenant-suspension check (§2.13) — a direct per-request DB read is intentional; no in-memory cache, no eventual-consistency window.
- Any client-side (frontend) special-casing of a 403-due-to-suspension response beyond surfacing the error message — no dedicated "your store is suspended" page/redirect, unless separately asked.

---

## 6. Open questions — flagged, NOT yet decided (discuss before building)

Unlike §2 (LOCKED) and §5 (explicit non-goals, decided-not-to-build), the items below are neither
— they're gaps found during design review that still need a decision. Don't silently resolve one
of these while implementing a phase; surface it and get an explicit decision first, the same way
every §2 item was decided.

1. ~~A tenant is never guaranteed to be bound to any subscription plan — old or new.~~ **RESOLVED
   2026-07-11 — see §2.15 + Phase B1.5.** Decided: (a) `ProvisionTenant` stays as-is, no auto-trial
   — a new tenant is package-inactive until its owner checks out; (b) a lapsed/missing package
   partially restricts `ActorAdmin` (Langganan-only) and fully blocks `ActorStaff`, computed live
   off `current_period_end`, no cron; (c) pre-existing tenants are backfilled (not grandfathered
   as fully exempt) via a one-time data migration.
   - **Residual sub-decision — RESOLVED during Phase B1.5 implementation, then REVISED 2026-07-14
     at the user's explicit request (production deploy was still pending, so this landed in the
     same not-yet-shipped migration — no production data was ever migrated under the old shape):**
     the backfill plan is no longer a free hidden `legacy-grandfather` (price 0, 20-year period).
     It's now a real paid plan named **"Premium Contributor"** — code `premium-contributor`,
     Rp1.700.000/year (`period_days=365`), still `is_active=0` (hidden from the tenant-facing
     picker — only ever assignable via this migration, never selectable at checkout). New
     `subscription_plans.renewal_only` column (migration `000015_subscription_billing`, `DEFAULT
     0`) is `1` for this plan only: a subscriber on a `renewal_only` plan can only ever renew the
     SAME plan — `Checkout()` rejects switching away from it to any other plan, and separately
     rejects anyone (including a fresh tenant) checking out into ANY hidden (`is_active=0`) plan
     that isn't already their own — enforced in
     `subscription/application.Service.validatePlanSwitch`, not just hidden in the UI. The
     backfill's initial `current_period_end` is **365 days** from migration time (not 20 years) —
     a legacy tenant gets one year on the house, then renews at the real price like a genuine
     Premium Contributor from then on. A "Perpanjang" (renew) button was added to
     `SubscriptionPage.tsx` — no such affordance existed before this revision for ANY plan. See
     migration `000018_subscription_legacy_backfill` and `docs/PRODUCTION_MIGRATION_PLAN.md` for
     the full analysis.

## 7. If you're an agent resuming this

1. `git status` / `git log --oneline -5` — confirm branch, confirm nothing unexpected landed.
2. Read §2 (locked decisions) fully before writing code.
3. **Check §3's checkboxes before touching §4 at all.** If any backend phase (B0–B6) is incomplete, that is the next task, full stop.
4. Follow the dependency diagram at the top of §3 for backend ordering (`B0`/`B1` first, independent of each other; `B1.5` needs only `B1`, independent of `B2`–`B5`; `B2` needs `B0`; `B3` independent; `B4` needs `B3`+`B2`; `B5` needs `B2`+`B3`; `B6` last). For frontend, `F0`→`F1`→`F2`→`F3` is a strict chain; `F4`/`F5` only need `F0` and can be done in parallel with `F1`–`F3` if splitting work; `F6` last. Run the relevant verify command after every phase.
5. Pay particular attention to §2.6/§2.7 (the two balance formulas, the claim/complete flow) when implementing B2 and F3's Penarikan page — this is the part of the design most likely to be subtly wrong if rushed. If in doubt, re-read the conversation's reasoning behind why a two-step flow exists before "simplifying" it back to one button. Similarly, §2.15's `ActorAdmin` (partial, allow-listed) vs `ActorStaff` (total) distinction is easy to accidentally flatten into one — don't let Phase B1.5 collapse it back into a single suspend-style block.
6. If a backend contract shape in §1/§3 looks wrong when you actually call it, trust the live Go source over this document — fix the code, then fix this file to match, before continuing.
7. If you add, remove, or reorder a phase, **update every cross-reference to it** (§1's mentions, other phases' "depends on" notes, §7 itself) in the same edit — don't let this document's own internal references drift out of sync the way earlier drafts did.
8. Check **§6 (open questions)** before starting work on subscription-related phases — if an item there is still undecided, that's a signal to stop and ask, not to guess an answer while coding.
9. Read **§1a (pre-flight code audit)** before starting `B1.5`, `F0`, `F2`, or `F5` specifically — those four phases had assumptions corrected by a code audit after this file was first drafted; the corrections are already folded into their checklists, but §1a explains *why* each checklist looks the way it does.
10. **Part 1 (§0–§8) is fully implemented** — there is no next phase to resume within it. If you're reading this because something in Part 1 regressed, §8's bug list is the first place to check whether it's a known-and-fixed issue resurfacing. **Part 2 (§9) is planned but not started** — if you're resuming work and Part 1's checkboxes are all ticked, §9.2's `PB0` is the next task. Read §9 in full before touching payment-module code; its own resume-guide is at §9.8.

---

## 8. Implementation verification log (2026-07-11/12)

All 15 phases (B0–B6, F0–F6) were implemented and verified end-to-end — real browser sessions
(Playwright) against a live backend + local MySQL, not just unit-level checks. Static checks
(`go build`/`go vet`/`go test`, `tsc --noEmit`, `eslint .`) all pass, but — consistent with §1a's
lesson that this kind of gap only surfaces by actually calling the code — several **real bugs**
were found only through E2E testing and fixed in place (not deferred):

1. **`platform/domain.Tenant` had no `json` tags at all** — serialized as `{"ID":...,"Name":...}`
   instead of camelCase, so the Tenant page rendered blank names/slugs/dates and a React key
   warning (since `id` was `undefined` for every row). Pre-existing in the already-shipped
   "Phase 1 done" code — never caught because no frontend had ever called `GET /platform/tenants`
   before Phase F3. Fixed: added tags.
2. **`subscriptionclient.Subscription` had no `planName`** — a subscriber on the hidden
   `legacy-grandfather` backfill plan (§2.15) showed "Paket tidak diketahui" on Langganan, because
   the tenant-facing `listPlans()` only returns `is_active=1` plans. Fixed: `Current()` now
   resolves the plan's name via `GetPlan` (not the active-only list) regardless of its active flag.
3. **The subscription-gate allowlist (§2.15) never actually matched anything but its own mount
   root** — chi's `RouteContext().RoutePattern()` collapses any sub-path under a `Route()`-mounted
   group into a wildcard leaf (`/api/v1/subscription/plans` reports as `/api/v1/subscription/*`),
   so `GET /subscription/plans` (and `/checkout`, `/invoices`) 402'd for a package-inactive owner
   even though they're supposed to be reachable. Fixed: match on `r.URL.Path` with prefix
   comparison instead — also more faithful to §2.15's own original wording (`GET/POST
   /subscription*`, a prefix).
4. **`GET /auth/me` was missing from that same allowlist** — found while fixing #3: `/auth/me`
   sits behind the identical `Authenticate` middleware as everything else, so a package-inactive
   owner's session-restore call would itself 402, and the frontend would treat that as an invalid
   session and log them out — instead of showing the locked Langganan shell §2.15 intends. Fixed:
   added to the allowlist.
5. **Tenant-facing `withdrawal.Repo.List` never mapped `processed_by`/`claimed_at`/`processed_at`/
   `rejected_reason`** — it had its own inline struct literal predating those columns (Phase B0),
   so a tenant's own Withdrawals page never showed a rejection reason even though the platform
   side (which used the newer `toDomain()` helper) did. Fixed: `List` now uses `toDomainList()`
   like every other read path in this module.
6. **Both `useAuthStore.login` and `usePlatformAuthStore.login` swallowed the backend's actual
   error message** on any login failure, always showing a generic "Email atau password salah" —
   so a suspended tenant's owner saw "wrong password" instead of "Toko Anda sedang dinonaktifkan.
   Hubungi platform." (§2.13), even though the backend already returns the right message. The
   tenant store's version pre-dates this feature; the platform store's version was a faithful
   copy of the same bug made during Phase F1. Fixed: both now surface `ApiError.message` when
   available, falling back to the generic string only for a non-`ApiError` (e.g. network) failure.

None of the above reopened a §2 LOCKED decision or changed scope — all six are corrections to
make already-decided behavior actually work as specified.

**What was walked end-to-end through the real UI** (not just curl/unit-level): both login pages
+ token isolation (§2.1, including simultaneous tenant+platform sessions in one browser and
logout-one-leaves-the-other-intact); all 7 Konsol Platform pages including tenant/plan/user CRUD;
the full claim → complete and claim → reject withdrawal flows, including the two-superadmin
claim-race scenario (§2.7) with a second real superadmin account; the Langganan page's plan-picker
→ QRIS checkout → manual status-check flow; the §2.15 subscription-lock guard (redirect + reduced
shell + banner, confirmed the backend 402 still fires even when the frontend's nav briefly shows
stale state after a hard reload — the enforcement boundary is server-side, not just cosmetic); and
the full §2.13 suspension lifecycle (fresh login rejected, an already-issued access token 403s on
its very next call, a still-valid refresh token is rejected, and reactivation restores access
immediately on the *same* pre-existing token with zero caching delay).

**Not exercised** (would need real Tripay/Midtrans sandbox credentials or a live SMTP account,
neither available in this environment): an actual (non-simulated) QRIS payment webhook callback,
and actual delivery of the withdrawal-submission notification email (its no-op-when-unconfigured
path, `mail.Config.Enabled() == false`, is what ran here — confirmed the tenant-facing response
never reveals whether a send was attempted, per §2.10, but the real `net/smtp` send path itself
was not hit against a live mail server).

### Post-implementation deep review (2026-07-12)

A follow-up point-by-point audit against every §2 LOCKED decision (two independent passes: a
direct re-read of the live code, plus a separate adversarial agent given only PLAN.md + the repo,
no prior context) confirmed everything above holds under closer scrutiny, **except one open,
not-yet-fixed finding**:

- **`withdrawal.Claim`'s claimable-balance check (§2.6) and its atomic status-transition `UPDATE`
  (§2.7) are two separate, non-transactional statements.** The atomic `WHERE status='pending'`
  update genuinely prevents double-claiming the *same* row, but does **not** prevent two
  *different* pending withdrawal rows for the same store from being claimed concurrently by two
  superadmins whose claimable-balance checks both run before either's `UPDATE` commits — each
  reads the same pre-claim balance, both pass, both succeed. Net effect: the store's balance can
  be over-committed beyond what §2.6's own text implies is guaranteed ("whichever gets claimed
  first locks in that amount... the rest fail at claim-time"). This requires two real superadmins
  clicking "Klaim" on two *different* requests for the *same* tenant within the same narrow race
  window — not caught by `go test`/`go vet`, and not exercised by this phase's manual claim-race
  test (which only tested two superadmins racing the *same* row, which IS correctly handled).
  A proper fix needs either a DB-level lock (`SELECT ... FOR UPDATE` on the store's withdrawal
  rows) spanning the balance check + the update, or a serializable transaction — complicated by
  `AvailableBalance` needing a live cross-module call to `salesclient` (QRIS revenue), which isn't
  itself inside a DB transaction today. **Decision (2026-07-12): accepted as a known risk, not
  fixed.** The blast radius is narrow (requires two superadmins to click "Klaim" on two different
  requests for the same tenant within the same race window — this app has a handful of
  superadmins, not a high-concurrency actor pool) and self-corrects to an auditable discrepancy
  (every claim is `processed_by`-attributed) rather than a security bypass or silently lost money.
  Revisit if the superadmin count or claim volume ever grows enough to make this realistically
  reachable.

### Master end-to-end scenario (2026-07-12)

A single continuous Playwright scenario (10 stages) walked every new feature in one coherent
tenant lifecycle, rather than testing each phase in isolation: platform onboards a fresh tenant
→ tenant is subscription-locked → tenant subscribes (checkout → simulated payment → unlock,
correct upgrade-only CTA) → tenant adds staff and staff can log in → QRIS revenue recorded and
visible on both tenant and platform sides → tenant submits withdrawals (over-limit rejected
client-side) → platform claims/completes one withdrawal (audit trail + balance drop correct)
→ tenant suspended mid-flow with a withdrawal still pending (login rejected, claim blocked per
§2.14, reject still allowed, reactivation restores access immediately) → platform Ringkasan
reconciliation figures verified by arithmetic → platform user management (self-deactivation
guard, password reset on another superadmin).

**Result: zero new product bugs.** All 3 failures hit while authoring the scenario were bugs in
the *test script's* Playwright selectors (ambiguous `text=`/`button:has-text` matches resolving
to the wrong one of several identically-labeled elements — e.g. a page header button vs. a modal
submit button both named "Tambah Staf"), not in the application. A separate final regression
pass over unrelated tenant pages (Dashboard, Produk, Kategori, Transaksi, Shift, Meja, Kas,
Statistik, Pengaturan, Penarikan, Langganan) using the original bootstrap admin account (still on
the hidden `legacy-grandfather` backfill plan) confirmed no collateral regressions and, as a
side-effect, re-confirmed the B6 legacy-backfill migration keeps that account's access unlocked.

---

## 9. Part 2 — Payment Gateway Multi-App Abstraction (planned 2026-07-12, COMPLETE — implemented and verified 2026-07-12)

### 9.0 Context and motivation

Today's `payment` module (`apps/api/internal/modules/payment/`) is a single-gateway, no-identity
wrapper: `NewClient` picks exactly one active provider (Tripay or Midtrans) from `config.Payment`
— loaded once from `.env` at process start — and returns one shared `apiClient` singleton, injected
into both existing consumers (`selforder`, `subscription`) and the webhook handler. `CreateCharge`'s
`storeID` parameter is accepted but explicitly unused. The only thing distinguishing "whose charge
is this" today is an ad-hoc convention owned by the `subscription` domain: it prefixes its own
order refs with `"sub_"` (`subscription/domain/subscription.go`); the composition-root webhook
dispatcher (`internal/app/webhook.go`) checks that prefix to decide whether to route an incoming
callback to `subscription` or fall back to `selforder`. This works for exactly two known, trusted,
in-process Go consumers — it has no formal registry, no way to configure gateway credentials
without editing `.env` and redeploying, and no capability surface beyond QRIS (the only channel
either existing consumer has ever needed).

Three separate problems motivate this part, all raised and analyzed in conversation on 2026-07-12
(full rationale lives in that conversation; this section carries only the decisions):

1. **Operational friction**: rotating a compromised Tripay key, fixing a typo'd credential, or
   switching sandbox→production today requires editing `apps/api/.env` on the server and
   restarting the process. A superadmin should be able to do this from Konsol Platform instead.
2. **No formal notion of "who is calling"**: the `"sub_"`-prefix trick is a private implementation
   detail of one consumer, not a reusable concept. It cannot scale to a third internal consumer
   without adding another bespoke prefix, and it has no story at all for an external caller.
3. **Narrow capability surface**: the module's public contract (`paymentclient.Client`) only knows
   how to create a QRIS charge, because that's all Elkasir itself has ever needed. A future SaaS
   product that wants to reuse this gateway might need virtual account, retail-outlet, or
   e-wallet channels that Elkasir never asked for — the contract should not be shaped entirely by
   Elkasir's own current usage.

**The one constraint that shapes every decision below, stated explicitly by the user:** there is
still exactly **one** Tripay/Midtrans merchant account — "one wallet." Registering more apps never
means provisioning more gateway credentials; it only ever adds rows to an internal registry used
for attribution and webhook dispatch. Money for every app funnels through the same account.

### 9.1 LOCKED design decisions

1. **One wallet, many registered apps.** `config.Payment`'s single active-provider/credential model
   is preserved in spirit (still exactly one active gateway at a time, globally) — it just moves
   from `.env`-at-boot to a DB-backed, superadmin-editable single row. Registering an "app" in the
   new registry (§9.1.3) never implies separate gateway credentials for that app.

2. **Gateway credentials move to the database; exactly one new secret stays in `.env`.** A new
   `payment_gateway_config` table (owned by `payment`, single logical row — `id` fixed/singleton or
   enforced-single-row by convention, not a multi-row table) holds the active provider plus its
   credentials (Tripay: API key, private key, merchant code, method; Midtrans: server key).
   Secrets are encrypted at rest (AES-256-GCM, Go stdlib `crypto/aes`+`crypto/cipher` — no new
   third-party dependency, consistent with this codebase's existing no-SDK philosophy for
   `mail`/`payment`) using a key from a **new** env var, `CONFIG_ENCRYPTION_KEY` — the one secret
   that necessarily still lives outside the database, since something must bootstrap the
   encryption. Every other payment credential (`TRIPAY_API_KEY`, `TRIPAY_PRIVATE_KEY`,
   `TRIPAY_MERCHANT_CODE`, `TRIPAY_QRIS_METHOD`, `MIDTRANS_SERVER_KEY`, `PAYMENT_PROVIDER`,
   `PAYMENT_ENV`) moves out of `.env` into this table. Konsol Platform's config form treats secret
   fields as **write-only** — a saved secret is never echoed back to the browser, only a masked
   placeholder (e.g. `••••••••1234`, last 4 chars) confirming one is set.

3. **New `payment_clients` table — the "app" registry, owned by `payment`.** Columns: `id` (ULID),
   `app_id` (human-readable, unique, e.g. `ELKASIR-SELFORDER`), `name`, `secret_hash` (bcrypt, same
   pattern already used for every password in this codebase — see `bootstrap/seed.go`), `kind`
   (`internal` | `external`), `callback_url` (nullable; unused for `kind=internal`, required for
   `kind=external` once §9.7's external API exists), `status` (`active`|`inactive`), timestamps.
   Never hard-deleted (same "deactivate, don't delete" precedent as `platform_users`, §2.9) — an
   app's history of charges must stay attributable. A migration seeds exactly two `kind=internal`
   rows at creation time, mirroring the `legacy-grandfather` seeding precedent (migration
   `000018_subscription_legacy_backfill`): `ELKASIR-SELFORDER` and `ELKASIR-SUBSCRIBE`.

4. **`CreateCharge` gains a required `appID` parameter, replacing the `"sub_"` prefix hack
   entirely.** New signature: `CreateCharge(ctx, appID, storeID, orderID string, amount int64)
   (Charge, error)`. The payment module records a **thin internal index** — not a business ledger
   (§2 of Part 1's "payment owns no ledger" principle is unchanged) — mapping `provider_ref`/
   `order_ref` to `app_id`, used only for webhook dispatch. `subscription/domain`'s `refPrefix`,
   `OrderRef`, and `OwnsRef` are deleted; the composition root's `subscriptionConsumer` interface
   (with its `OwnsRef` method) in `internal/app/webhook.go` is deleted along with the file itself
   (its logic moves per decision 5 below). Both existing call sites (`selforder/application/
   service.go`, `subscription/application/service.go`) are updated **in the same phase** so
   `go build` stays green throughout — this is not a two-step migration with a transitional dual
   path.

5. **Webhook dispatch moves from the composition root into the payment module itself, driven by
   the registry instead of hardcoded branching.** `payment` gains its first HTTP route ever
   (`payment/presentation/`) — still exactly **one** route, still the same URL
   (`/api/v1/webhooks/payment`, unchanged — nothing needs re-registering with Tripay/Midtrans).
   Internal (`kind=internal`) dispatch stays an in-process Go call: `app.go` builds a
   `map[string]webhookConsumer` keyed by `app_id` from the same two Go interfaces that exist today
   (`selforder`, `subscription` — their `ApplyWebhookEvent` method signature is **unchanged**, they
   are not aware this refactor happened). The payment module looks up which `app_id` owns an
   incoming event's `order_ref` via its own index table (decision 4) and calls that consumer —
   no string-prefix sniffing anywhere. External (`kind=external`) dispatch is designed for but
   **not built** in this part — see §9.7.

6. **Config changes take effect on next use, via explicit invalidation — no polling, no background
   refresh loop.** Consistent with Part 1's established "compute live, no caching" philosophy
   (§2.13/§2.15): the config-update HTTP handler explicitly triggers a rebuild of the underlying
   gateway client (`apiClient`) after a successful write. There is no TTL cache and no periodic
   poll of the config table.

7. **One-time env→DB migration path for existing deployments.** On first boot after this part
   ships, if `payment_gateway_config` has no row, the app reads the legacy `TRIPAY_*`/
   `MIDTRANS_*`/`PAYMENT_PROVIDER`/`PAYMENT_ENV` env vars once, encrypts and inserts them as the
   initial DB row, and logs that it did so. After that first boot, the env vars are dead — present
   or absent, they're never read again. This is a one-way migration, not a permanent dual-source
   fallback (don't build a "DB row missing → fall back to env forever" path; that would leave two
   sources of truth indefinitely).

8. **The contract expands beyond QRIS-only, but conservatively — three additions, nothing more
   speculative. Channel enum decided 2026-07-12: exactly `ChannelQRIS` and `ChannelVA` (virtual
   account) for now — retail (Alfamart/Indomaret) and e-wallet are explicitly deferred, not built,
   not stubbed with an error case; add them as a follow-up if/when actually needed, same
   conservative-expansion philosophy this whole decision already follows.**
   - `CreateChannelCharge(ctx, appID, storeID, orderID string, amount int64, channel Channel,
     opts ChannelOptions) (Charge, error)` — a general charge-creation method parameterized by
     channel (`ChannelQRIS` | `ChannelVA`). `ChannelOptions` for VA carries whatever Tripay's VA
     creation call needs beyond amount/orderID (at minimum a bank code) — **specific VA bank codes
     (BCA, Mandiri, BNI, etc.) are not hardcoded in the `Channel` enum or anywhere in this contract**
     — they're discovered at runtime via `ListChannels()` below, which reports whatever bank VA
     codes are actually enabled on the Tripay account. Adding/removing a bank in Tripay's own
     dashboard should never require a code change here. The existing `CreateCharge` **stays**,
     unchanged in behavior, as a QRIS-specific convenience wrapper (`CreateCharge(...) ==
     CreateChannelCharge(..., ChannelQRIS, nil)`) — neither existing consumer needs to change how
     it calls this.
   - `ListChannels(ctx context.Context) ([]ChannelInfo, error)` — reports which channels are
     currently enabled/configured, mirroring what the gateway's own channel-listing API exposes.
     Lets a future consumer discover capability dynamically instead of hardcoding an assumption.
   - `CheckStatus(ctx context.Context, providerRef string) (ChargeStatus, error)` — a pull-based
     status check, independent of the webhook push path. Useful when a webhook is delayed or lost
     — a real gap in the current design (today, a missed webhook means a charge's paid status is
     never discovered by any other means). This closes that gap for existing consumers too, not
     just future ones.
   - **Explicitly excluded from this list** (see §9.4): refund/void, payout/disbursement, and
     recurring/subscription-native billing. These involve money moving *out* or automating what
     Part 1 deliberately made manual (§5 of Part 1: "Automatic bank-transfer API integration —
     processing stays manual") — bigger in scope and risk than "wrap what's already inbound," and
     not something either current consumer or the user has asked for yet.

9. **The registry's `secret` is stored and hashed from day one, but unenforced for `kind=internal`
   rows.** Internal consumers call `CreateCharge`/`CreateChannelCharge` etc. as direct in-process
   Go function calls — there is no network hop to authenticate across, so requiring a secret today
   would be theater. The column exists now so the data model doesn't need to change later; the
   actual HMAC/signature verification is real work deferred to §9.7's external API, at which point
   it starts being enforced for `kind=external` rows only.

10. **Decided 2026-07-12 — superadmin routes for gateway config + app registry live under
    `/platform/*`, owned by the `platform` module — not new routes on `payment` itself.** This
    mirrors the existing precedent exactly: `platform` already consumes `withdrawalclient` and
    `platformuserclient` as contracts and exposes the actual HTTP routes/UI for them (§1/MODULE_MAP
    of Part 1) — it never lets `withdrawal`/`platformuser` grow their own superadmin-facing routes.
    `payment` gains new contract methods (`GetConfig`/`UpdateConfig` on `paymentclient.Client`, plus
    a registry-management surface — `ListApps`/`CreateApp`/`ResetAppSecret`/`SetAppStatus`, either
    added to `paymentclient.Client` or as a small second contract if that keeps the interface
    cleaner, implementer's call) but exposes **no HTTP routes of its own for these** — same
    contracts-only shape `platformuser` already uses (no `presentation/` package for this surface,
    even though `payment` *does* get a `presentation/` package for the webhook endpoint per
    decision 5 — those are two different concerns and don't have to share a package). `platform`'s
    application service calls these new contract methods and its own presentation layer exposes
    `GET/PUT /platform/payment-config` and `GET/POST /platform/payment-clients` (+ sub-actions),
    all gated `RequireActor(ActorPlatform)` like everything else `platform` exposes.

11. **Decided 2026-07-12 — `kind=external` app registration is allowed now, before §9.7's external
    API exists, with an explicit "not yet active" signal.** The "Aplikasi Terdaftar" page (§9.3
    `PF1`) lets a superadmin register an external app's `APP-ID`+`SECRET` immediately — it doesn't
    wait for §9.7. Every `kind=external` row displays a persistent badge — *"API eksternal belum
    tersedia"* — until §9.7 actually ships, so a superadmin registering one now isn't misled into
    thinking a partner can already call in.

### 9.2 Backend implementation — `PB0`–`PB4`

Dependency order: `PB0` (schema) first. `PB1` (contract signature changes) needs `PB0`'s tables to
exist for the index-recording. `PB2` (config storage + hot-reload) is independent of `PB1`, can be
done in parallel. `PB3` (registry-driven webhook dispatch) needs both `PB1` (appID tagging) and
`PB2` (nothing structural, but sequenced after for a cleaner diff). `PB4` closes out with docs +
verification. Run `go build ./... && go vet ./... && go test ./...` after each phase, same
discipline as Part 1.

#### Phase PB0 — Migrations: registry + config tables

- [x] New migration: `payment_clients` table per §9.1.3. Unique index on `app_id`.
- [x] New migration: `payment_gateway_config` table per §9.1.2 (encrypted credential columns +
      active-provider column). Enforce single-row via application logic (upsert by a fixed known
      id), not a DB constraint — simpler, and consistent with how `settings` already does
      effectively-single-row-per-store config.
- [x] Seed migration: insert `ELKASIR-SELFORDER` and `ELKASIR-SUBSCRIBE` (`kind=internal`,
      `status=active`) — same migration-seeds-a-reference-row precedent as `legacy-grandfather`.
- [x] New env var `CONFIG_ENCRYPTION_KEY` added to `apps/api/.env` (real value) and `.env.example`
      (placeholder). Document the one-time env→DB migration (§9.1.7) inline in this phase's notes.
- [x] `go build ./... && go vet ./...` clean (no Go code changes yet, just schema — this step is
      really "migrations apply cleanly").

#### Phase PB1 — Contract changes: `appID` parameter + broader channel surface

Depends on `PB0`.

- [x] `paymentclient.Client.CreateCharge` gains `appID` (§9.1.4). Update both call sites in the
      same phase: `selforder/application/service.go` passes `"ELKASIR-SELFORDER"`,
      `subscription/application/service.go` passes `"ELKASIR-SUBSCRIBE"`.
- [x] Delete `subscription/domain`'s `refPrefix`/`OrderRef`/`OwnsRef` and the
      `subscriptionConsumer` interface in `internal/app/webhook.go` — `subscription`'s order ref
      goes back to being a raw invoice id, no prefix.
- [x] New index table (or reuse `payment_clients` with a join table — implementer's choice, but
      document whichever is chosen) recording `order_ref → app_id` per charge, written inside
      `CreateCharge`/`CreateChannelCharge`.
- [x] Add `Channel`/`ChannelInfo`/`ChannelOptions`/`ChargeStatus` types and
      `CreateChannelCharge`/`ListChannels`/`CheckStatus` to `paymentclient.Client` (§9.1.8).
      `CreateCharge` becomes a thin wrapper over `CreateChannelCharge` with `channel=qris`.
- [x] Wire `ChannelVA` for real in the Tripay gateway (`payment/infrastructure/tripay.go`) —
      Tripay's VA creation call, mapped through the same normalized `qrResult`-style shape (or a
      new `vaResult` if the response shape genuinely doesn't fit — VA returns an account
      number/bank code, not a QR image, so don't force it through `qrResult` if that means lying
      about unused fields). `ListChannels()` reflects whichever VA bank codes are actually enabled
      on the Tripay account — don't hardcode a bank list (§9.1.8). Midtrans's equivalent (if/when
      Midtrans is the active provider) is out of scope for this pass unless the user is actually
      running Midtrans in production — confirm which provider is live before spending time on
      Midtrans VA specifically.
- [x] `go build ./... && go vet ./...` clean.

#### Phase PB2 — DB-backed gateway config + hot-reload

Independent of PB1 — can be built in parallel.

- [x] Repository for `payment_gateway_config`: encrypt on write (AES-256-GCM,
      `CONFIG_ENCRYPTION_KEY`), decrypt on read.
- [x] `apiClient` gains an explicit rebuild path (§9.1.6) — the config-update handler calls it
      directly after a successful write; no cache, no poll.
- [x] One-time env→DB migration on first boot (§9.1.7): if the config table is empty, read the
      legacy env vars once, encrypt, insert, log that the migration ran. After that, the legacy
      env vars are never read again by this module.
- [x] `paymentclient.Client` gains `GetConfig`/`UpdateConfig` methods (decision §9.1.10) — no
      routes on `payment` itself for this.
- [x] `platform` module: constructor gains this new contract surface (5th contract, alongside
      `subscriptionclient`/`salesclient`/`withdrawalclient`/`platformuserclient`); new routes
      `GET`/`PUT /platform/payment-config`, gated `RequireActor(ActorPlatform)` — secret fields
      write-only on the way in, masked on the way out.
- [x] `go build ./... && go vet ./...` clean.

#### Phase PB3 — Registry-driven webhook dispatch, moved into `payment`

Depends on PB1 (appID tagging exists to look up) and PB2 (sequenced after, no hard dependency).

- [x] New `payment/presentation/` package — the module's first-ever HTTP route,
      `POST /webhooks/payment` (same URL as today — nothing to re-register with the gateway).
- [x] Dispatch logic moves from `internal/app/webhook.go` (deleted) into this new presentation
      handler: verify → parse → idempotency-check (unchanged from today) → look up `app_id` from
      the order-ref index (PB1) → look up the registered Go consumer for that `app_id` in an
      in-process map built once in `app.go` → call `ApplyWebhookEvent` (consumer interface
      unchanged, per §9.1.5).
- [x] `paymentclient.Client` (or a small second contract, implementer's call per §9.1.10) gains
      `ListApps`/`CreateApp`/`ResetAppSecret`/`SetAppStatus` for the registry (§9.1.3) — again no
      routes on `payment` itself.
- [x] `platform` module: new routes `GET/POST /platform/payment-clients` + sub-actions
      (reset-secret, set-status), gated `RequireActor(ActorPlatform)` (§9.1.10). Secret is
      returned **once**, on creation only (§9.1.3) — never again afterward. The two seeded
      `kind=internal` rows reject deactivation at the application layer (§9.3 `PF1`) — they back
      live self-order/subscription traffic.
- [x] `go build ./... && go vet ./...` clean.

#### Phase PB4 — Docs + backend verification

Depends on PB0–PB3.

- [x] Update `knowledge/MODULE_MAP.md` — the "One shared payment webhook, two consumers" section
      (currently describing the `"sub_"`-prefix convention) needs a full rewrite to describe
      registry-driven dispatch instead. Update `knowledge/DATABASE_GUIDE.md`, `docs/DB_SCHEMA.md`
      for the two new tables.
- [x] `go build ./... && go vet ./... && go test ./...` clean — gate before `PF0` starts.
- [x] Manual smoke test: rotate the active Tripay credential via the new config route (curl, no
      frontend yet) → confirm the *next* charge uses the new credential without a process restart.
      Create a self-order charge and a subscription charge → confirm both are tagged with the
      right `app_id` in the new index → simulate a webhook for each → confirm both still route to
      the correct consumer exactly as before this refactor (this is the regression check that
      matters most — Part 1's whole withdrawal/subscription flow depends on this dispatch working).
      **Done 2026-07-12, all green:** confirmed the one-time env→DB migration populated
      `payment_gateway_config` correctly on first boot; a real subscription checkout tagged
      `ELKASIR-SUBSCRIBE` in `payment_charge_apps`; a real self-order QRIS checkout tagged
      `ELKASIR-SELFORDER`; a signed Tripay webhook simulated for each correctly dispatched via the
      registry (subscription period extended; self-order stock decremented + transaction
      recorded) — zero regression from the old prefix-based dispatch. `GET/PUT
      /platform/payment-config` and the full `/platform/payment-clients` CRUD (create → secret
      shown once → reset-secret → deactivate) all verified; the internal-row deactivation guard
      correctly rejected `ELKASIR-SELFORDER`. One real bug found + fixed here: every new contract
      struct (`AppInfo`, `GatewayConfig`, `Charge`, etc.) was missing `json` tags, serializing as
      PascalCase instead of this codebase's camelCase convention — same bug class as Part 1's
      `platform/domain.Tenant` finding (§8). Fixed before any frontend code was written against it.

### 9.3 Frontend implementation — `PF0`–`PF2`

Only after §9.2 is fully green, same ordering discipline as Part 1.

#### Phase PF0 — Konsol Platform: "Konfigurasi Pembayaran" page

- [x] New page under Sistem (sidebar) — active provider selector + credential fields, secret
      fields rendered write-only (masked placeholder, blank on load, only sent if the user typed a
      new value — never round-tripped from the server per §9.1.2).
- [x] Verify manually: change a credential, confirm a subsequent checkout (self-order or
      subscription) actually uses it.

#### Phase PF1 — Konsol Platform: "Aplikasi Terdaftar" page

- [x] Table: Nama Aplikasi | APP-ID | Jenis (Internal/Eksternal) | Status | Dibuat. The two seeded
      rows (`ELKASIR-SELFORDER`, `ELKASIR-SUBSCRIBE`) show as `kind=internal`, not deletable, not
      deactivatable (they back live production traffic — deactivating them would break self-order
      or subscription checkout entirely; the UI should prevent this specifically, not just rely on
      "don't click that").
- [x] "Daftarkan Aplikasi Baru" — creates a `kind=external` row, shows the generated `APP-ID` +
      `SECRET` **once** in a dismissible panel with a copy button and an explicit "you will not see
      this secret again" warning, matching the established one-time-reveal pattern already used
      elsewhere for credentials in this ecosystem. Allowed now, ahead of §9.7 (§9.1.11) — every
      `kind=external` row (including one just created) shows a persistent, non-dismissible badge
      *"API eksternal belum tersedia"* until §9.7 ships.
- [x] "Reset Secret" and "Nonaktifkan" actions for `kind=external` rows only.
- [x] Verify manually: register a test external app, confirm the secret only ever displays once
      (reload the page, confirm it's gone from the UI — still present, hashed, in the DB).

#### Phase PF2 — Final verification pass

- [x] `tsc --noEmit` clean, `eslint .` no new errors.
- [x] Full regression of Part 1's existing flows that depend on payment: self-order QRIS checkout
      end-to-end, subscription checkout end-to-end, both webhook paths — confirm nothing in Part 1
      regressed from this refactor (this is the highest-risk area of Part 2, since it touches code
      three existing, already-shipped, already-verified flows depend on).
- [x] Update this section's checkboxes, or note explicitly what's left and why.

**Verified 2026-07-12, real browser (Playwright), all green:** logged in as superadmin, walked
"Konfigurasi Pembayaran" (masked secret placeholders render correctly; saving without touching
secret fields preserves them — no accidental wipe) and "Aplikasi Terdaftar" (both seeded internal
apps visible, their action menu correctly disabled; registered a real external test app, secret
reveal modal showed exactly once and never reappeared after closing; row appeared in the table).
Zero console errors. A full regression sweep of 7 other Konsol Platform pages + 12 tenant pages
(dashboard, self-order incoming, withdrawals, subscription, etc.) found zero collateral damage.
Test artifacts (the external test app row) cleaned up after verification.

### 9.4 Explicit non-goals for Part 2 (don't build unless separately asked)

- Refund/void, payout/disbursement automation, or recurring/subscription-native billing via the
  gateway (§9.1.8) — bigger in scope than "wrap inbound capability," not asked for.
- Automating any part of the `withdrawal` module's manual claim→complete flow using a new gateway
  capability — Part 1's §5 non-goal ("Automatic bank-transfer API integration — processing stays
  manual") is unchanged by this part.
- Running two gateways simultaneously, or per-app gateway credentials — still exactly one active
  provider globally (§9.1.1).
- The actual external-facing HTTP API for third-party SaaS callers — designed for (§9.1.5,
  `kind=external` exists in the data model) but not built; see §9.7.
- Enforcing/verifying the registry `secret` for `kind=internal` rows (§9.1.9) — there's no network
  boundary to authenticate across for in-process callers.
- A polling or scheduled-refresh mechanism for gateway config — explicitly rejected in favor of
  explicit invalidation-on-write (§9.1.6).

### 9.5 Open questions — all resolved as of 2026-07-12

All three original items here are now decided — kept below (struck through) for the same
traceability reason Part 1's §6 keeps its resolved item, not deleted outright:

1. ~~Exact channel enum for `CreateChannelCharge`/`ListChannels`.~~ **RESOLVED 2026-07-12 — see
   §9.1.8: exactly `ChannelQRIS` + `ChannelVA` for now.** Retail/e-wallet deferred, not stubbed.
2. ~~Where the new superadmin routes for config/registry live.~~ **RESOLVED — see §9.1.10:
   `/platform/*`, owned by `platform`.**
3. ~~Whether `kind=external` app creation should be gated behind §9.7's external API existing.~~
   **RESOLVED — see §9.1.11: allowed now, gated by a persistent "belum aktif" badge.**

Nothing currently blocks starting `PB0`.

### 9.6 How Part 2 changes already-shipped Part 1 code

This part is not additive-only — it modifies files Part 1 already shipped and verified:

- `internal/app/webhook.go` — **deleted**, logic moves into `payment/presentation/` (§9.1.5).
- `internal/app/app.go` — payment module construction gains the config-hot-reload wiring and the
  `app_id → webhookConsumer` map; the webhook route registration moves from here into
  `payment.module.go`'s own route mounting.
- `subscription/domain/subscription.go` — `refPrefix`/`OrderRef`/`OwnsRef` deleted (§9.1.4).
- `subscription/application/service.go` and `selforder/application/service.go` — their
  `CreateCharge` call sites gain the new `appID` argument (§9.1.4).
- `knowledge/MODULE_MAP.md`'s "One shared payment webhook, two consumers" section — rewritten
  (§9.2 `PB4`), since it currently documents the prefix convention this part removes.
- `apps/api/.env` / `.env.example` — `TRIPAY_*`, `MIDTRANS_*`, `PAYMENT_PROVIDER`, `PAYMENT_ENV`
  become dead after the one-time migration (§9.1.7); `CONFIG_ENCRYPTION_KEY` is added.

None of this reopens a Part 1 §2 LOCKED decision — Part 1's business rules (balance formulas,
claim/complete flow, suspension/subscription gates) are untouched; only the payment module's own
internals and its two call sites change shape.

### 9.7 Explicit scope boundary: internal groundwork now, external API later

Everything in §9.1–§9.3 is internal groundwork: a real registry, DB-backed config, appID-tagged
charges, registry-driven dispatch, a broader-but-still-inbound-only contract. **Deliberately
excluded from this part**, sequenced as a separate, later initiative once §9.2–§9.3 ship and are
verified against the two real existing consumers:

- An actual external-facing HTTP API (e.g. `POST /api/v1/external/payments/charge`) that a
  separate SaaS product could call over the network.
- Real enforcement of the `kind=external` secret (HMAC-signed requests or bearer-token auth).
- Outbound webhook relay — Elkasir forwarding a verified gateway callback to an external app's
  registered `callback_url`, signed with that app's secret.
- Rate limiting, per-app usage visibility/billing, and a security review appropriate for accepting
  traffic from outside this codebase's trust boundary.

Building this prematurely — before the internal foundation is solid and proven against real,
already-shipped traffic (self-order, subscription) — would mean designing an external contract with
no real external caller to validate it against. Revisit once §9.2–§9.3 are done and stable.

### 9.8 If you're an agent resuming Part 2

**Part 2 is fully implemented and verified (§9.2/§9.3's checkboxes + verification notes)** — there
is no next phase to resume within it. If you're reading this because something in Part 2
regressed, the verification notes at the end of `PB4` and `PF2` are the first place to check
whether it's a known-and-fixed issue resurfacing. If you're picking up the NEXT initiative:

1. §9.7 is the next real scope — the external-facing HTTP API, real secret enforcement, and
   outbound webhook relay for `kind=external` apps. Nothing in §9.1–§9.6 needs revisiting to start
   it; read §9.7 itself for the boundary of what's already done vs. what that work adds.
2. Confirm both Part 1 (§0–§8) and Part 2 (§9.2/§9.3) are still fully green
   (`go build && go vet && go test`, `tsc --noEmit`) before starting anything new — don't build on
   a broken base.
3. Read §9.1 (locked decisions) in full first — same discipline as §2 for Part 1 — since §9.7's
   work extends this same registry/dispatch design rather than replacing it.
4. If you add, remove, or reorder a phase within §9, update every cross-reference to it within §9
   (and §9.6's file-level change list, if the change affects which files are touched) in the same
   edit — same discipline §7 already established for Part 1.

---

## 10. Part 3 — External Payment API (planned 2026-07-12, COMPLETE — implemented and verified 2026-07-12)

### 10.0 Context and motivation

Part 2 (§9) built the internal groundwork — a registry (`payment_clients`), DB-backed config,
`appID`-tagged charges, registry-driven webhook dispatch — but deliberately stopped short of
§9.7: an actual HTTP surface a *separate* SaaS product (running in its own process, its own
codebase, on someone else's server) could call over the network. That's what this part designs.

The core problem this part must solve, precisely: a `kind=external` row in `payment_clients`
today is just a database row with an `app_id` and a hashed secret — nothing can authenticate
against it, nothing can create a charge on its behalf, and nothing relays a paid-webhook back to
it. Three concrete capabilities are needed, all reusing Part 2's registry rather than building a
second one:

1. A way for an external caller to prove "I am `app_id` X" over HTTP (today, `secret_hash` is
   bcrypt — one-way, fine for verifying a caller *presented* the right secret, but that's the
   only thing bcrypt can do).
2. Routes that let an authenticated external caller create a charge and check its status,
   scoped so it can only ever act as itself (never impersonate another registered app).
3. A way for Elkasir to prove to the *external app* that a webhook relay really came from
   Elkasir — which surfaces a real cryptographic tension resolved in §10.1.6 below.

### 10.1 LOCKED design decisions

1. **Reuse the existing JWT/auth infrastructure — a 4th `Actor`, not a parallel auth scheme.**
   `authcontract.Actor` gains `ActorApp = "app"`, alongside the existing `ActorAdmin`/
   `ActorStaff`/`ActorPlatform` (`auth/contracts/auth.go`). `Principal.SubjectID` holds the
   `payment_clients.id` (the row's ULID) — exactly the same convention already used for every
   other actor (`SubjectID` = `platform_users.id` for `ActorPlatform`, etc.). `Principal.StoreID`
   and `Principal.Role` stay empty for `ActorApp` — an external app has no store and no role
   tiers, mirroring how `ActorPlatform` already leaves `StoreID` empty. **No new field on
   `Principal`** — this reuses the shape as-is.

2. **`auth` gains its own narrow, read-only login-lookup query directly against
   `payment_clients`** — the SAME precedented pattern already used for `platform_users`/
   `admin_users`/`staff` (`knowledge/DATABASE_GUIDE.md`: "`auth` separately keeps its own narrow
   login-lookup queries against the same table, same split already used for
   admin_users/staff"). CRUD ownership of `payment_clients` stays 100% with `payment` (via
   `platform`'s passthrough, §9.1.10) — `auth` only ever reads `id`, `app_id`, `secret_hash`,
   `status` for the single purpose of issuing a token. No contract call to `paymentclient.Client`
   for this — same reasoning as why `auth` doesn't call a contract to check `admin_users`/`staff`
   login either.

3. **New endpoint `POST /api/v1/auth/app/token`** — a client-credentials exchange, NOT a
   session-style login: body `{appId, secret}` → `{accessToken, expiresIn}`. **No refresh token
   issued for `ActorApp`** — `refresh_tokens.actor` enum is untouched by this part. Re-running the
   client-credentials exchange when a token expires is trivial for a machine caller (unlike a
   human session, where seamless refresh matters); adding a refresh-token dance here would be
   unjustified complexity for no real benefit. New config `JWT.AppTokenTTL` (default 1 hour,
   distinct from the human `JWT.AccessTTL`/`JWT.RefreshTTL` pair) — app tokens are short-lived
   and cheap to reissue, not tuned for a human's session-continuity expectations.

4. **Live status check on every authenticated request, not just at token issuance — mirrors the
   exact philosophy already locked in §2.13 (tenant suspension) and §2.15 (subscription gate):
   "no caching, no eventual-consistency window."** `auth`'s middleware, for `ActorApp` principals
   specifically, re-reads `payment_clients.status` on every request (same narrow, precedented
   read as decision 2 — "same justification class as `settings`/`platform`'s existing exceptions,
   just for a status flag," to quote §2.14's own wording for the equivalent `stores.status`
   check). **Deactivating an external app immediately revokes ALL its access — even an
   already-issued, unexpired token stops working on its very next call.** This is the same
   guarantee Part 1 already gives tenants and Part 2 already gives the payment config; Part 3
   just extends it to a 4th actor type instead of inventing a weaker story for this one.

5. **`storeID` is empty for genuinely external charges.** `paymentclient.Client.CreateCharge`/
   `CreateChannelCharge` already accept a `storeID` parameter that the gateway itself never uses
   (documented in `payment/infrastructure/client.go` since Part 1: "kept as part of the contract
   because a future provider might need per-tenant identity"). A true external SaaS product has
   no Elkasir store at all — its charges pass `storeID=""`. Nothing about the existing contract
   needs to change for this; it already tolerates an empty value structurally.

6. **The most important decision in this part — how ONE secret serves TWO different
   cryptographic purposes, resolved by storing it in TWO forms, not by using two different
   secrets.** An external app needs exactly one secret value, shown once at creation (already
   built in Part 2, unchanged) — but Elkasir needs to use that secret in two directions that
   have opposite requirements:
   - **Incoming**: verify the external app's token request presented the right secret. A
     one-way bcrypt hash is correct and sufficient here (Part 2 already stores this —
     `payment_clients.secret_hash`).
   - **Outgoing**: sign a webhook relay payload so the external app can verify it really came
     from Elkasir (HMAC-SHA256 over the payload, using the shared secret as the HMAC key). This
     is the tension: **HMAC requires Elkasir to recover the plaintext secret** — a bcrypt hash is
     one-way by design and cannot be used to compute a valid HMAC. Part 2's `secret_hash` alone
     is structurally insufficient for this second purpose.

   **Resolution:** at creation time (and at reset-secret time), store the SAME plaintext secret
   in **two representations** — the existing bcrypt `secret_hash` (unchanged, still used for
   incoming-auth verification, keeping that path's one-way safety property) **plus a new,
   separately-encrypted `secret_enc` column** (AES-256-GCM, reusing the exact same
   `CONFIG_ENCRYPTION_KEY`-derived helper already built for gateway credentials in Part 2 — zero
   new crypto code) used ONLY at webhook-relay time to recover the plaintext for HMAC signing.
   Rejected alternatives and why: (a) *a single reversibly-encrypted secret used for both
   directions* — simpler, but weakens the incoming-auth path's safety property for no benefit,
   since incoming auth never needed reversibility in the first place; (b) *two independent
   secrets (an "auth secret" and a separate "signing key")* — avoids touching decision's
   reasoning above but doubles what the user has to generate/manage/rotate for no real gain, and
   would require a second one-time-reveal UI in the already-shipped "Aplikasi Terdaftar" page.
   Storing the SAME plaintext twice, once each way, gets the correct property for each direction
   **and** needs zero frontend changes (§10.1.7). `secret_enc` is NULL for `kind=internal` rows,
   same convention as `secret_hash`.

7. **No frontend changes in this part.** Because decision 6 keeps a single user-facing secret
   value (shown once, exactly as Part 2 already built), the existing "Aplikasi Terdaftar" page
   needs no new UI — `CreateApp`/`ResetAppSecret` in `payment/infrastructure` just populate one
   more column from the same plaintext they already have in hand at creation time. Part 3 is
   backend-only.

8. **Route namespace: `payment/presentation` owns these routes directly — NOT funneled through
   `platform`.** This is a deliberately different call from §9.1.10 (which put superadmin-facing
   config/registry routes under `platform`, since `platform` is the superadmin's one entry
   point). `ActorApp` is not a superadmin at all — it's `payment`'s own, genuinely new caller
   type, with no other natural home. New routes, all under `payment/presentation` (which already
   owns `/webhooks/payment` since Part 2):
   ```
   POST /api/v1/auth/app/token              (owned by `auth`, per decision 3 — listed here for completeness)
   POST /api/v1/external/payments/charges              body {orderRef, amount, channel, channelOptions}
   GET  /api/v1/external/payments/charges/{orderRef}/status
   GET  /api/v1/external/payments/channels
   ```
   All three `/external/payments/*` routes gated `RequireActor(ActorApp)`. The charge-creation
   and status routes resolve `appID` from `principal.SubjectID` (looking up the `payment_clients`
   row `payment` already has a query for, `GetPaymentClientByID`, built in Part 2) — an app can
   never pass a different app's identity in the request body; there is no such field to pass.

9. **Idempotency reuses `payment_charge_apps`' existing unique `order_ref` — no new table.** The
   external caller supplies `orderRef` (their own idempotency key, exactly like Tripay's own
   `merchant_ref` convention they're likely already familiar with). Retrying with the SAME
   `orderRef` hits `payment_charge_apps`' existing PRIMARY KEY and fails with `409 Conflict`
   (`httpx.Conflict`), whose message points the caller at the status-check endpoint instead of
   silently double-charging. Retrying with a NEW `orderRef` is, correctly, a new charge. This
   needed no new schema — the uniqueness Part 2 already built for dispatch purposes turns out to
   double as the idempotency guarantee for free.

10. **Outbound webhook relay: fire-and-forget, exactly one attempt, no delivery-log table.**
    When `payment`'s `Dispatch` (§9.1.5) resolves an incoming gateway webhook's `app_id` to a
    `kind=external` row (instead of a registered in-process `WebhookConsumer`), it spawns a
    goroutine that POSTs a signed JSON payload (`{eventId, orderRef, paid, timestamp}`, header
    `X-Elkasir-Signature: sha256=<hex HMAC>`, key = the `secret_enc`-recovered plaintext from
    decision 6) to the app's `callback_url`, and returns immediately — the goroutine's outcome
    never blocks or affects Elkasir's own `200 OK` back to Tripay/Midtrans (same fire-and-forget
    philosophy already locked for the withdrawal-notification email in Part 1, §2.10: "must
    never block, slow down, or fail" the thing that triggered it). On failure, `slog.Warn` only —
    no retry, no persisted delivery log, matching that same email precedent's lightness. The
    external app is expected to implement its own polling fallback via
    `GET /external/payments/charges/{orderRef}/status` for the rare case a relay attempt is lost
    (documented in the API reference, decision 12) — this is exactly the same self-healing shape
    §9.1.8 already gave internal consumers via `CheckStatus`, just used by a different kind of
    caller.

11. **Basic rate limiting, in-memory, no new infrastructure — thresholds decided 2026-07-12:
    60 requests/minute per `app_id` on the three `/external/payments/*` routes, 10 requests/minute
    per source IP on `POST /auth/app/token`.** A simple fixed-window counter, living in the
    `payment` module's presentation layer (charge/status/channel routes) and `auth`'s (token
    route) respectively. Exceeding the limit returns `429` via the already-existing (currently
    unused) `httpx.RateLimited` helper. No Redis, no distributed limiter — this is a
    single-process monolith; an in-memory map is sufficient and consistent with this codebase's
    whole "self-hosted, no extra infra" philosophy already established for `mail` and `payment` in
    earlier parts. Generous enough for normal integration traffic (roughly one charge per second
    sustained), tight enough to blunt an obvious brute-force/abuse attempt; revisit if a real
    integration's legitimate volume ever needs more.

12. **A written API reference IS in scope for this part — unlike Part 1/2's own precedent of
    deferring `openapi.yaml` updates until a real frontend consumer exists.** That precedent's
    reasoning ("a speculative OpenAPI entry would be unverifiable") doesn't transfer here: the
    "consumer" of THIS contract is, by definition, code Elkasir doesn't own and can't wait to
    exist before documenting — an external integrator has no other way to discover the contract
    shape at all. **Decided 2026-07-12: a dedicated `docs/EXTERNAL_PAYMENT_API.md`**, not an
    `openapi.yaml` addition — human-readable, can carry prose explanation + code examples (not
    just schema), and keeps `openapi.yaml` scoped to what it already is (the internal
    frontend↔backend contract) without mixing in a genuinely public-facing surface that has
    different versioning/stability expectations. `EB5` produces this file, covering every route,
    the signature-verification algorithm for relayed webhooks, error codes, and the
    idempotency/retry behavior from decisions 9/10.

### 10.2 Backend implementation — `EB0`–`EB5`

Dependency order: `EB0` (schema) first. `EB1` (auth/token issuance) needs `EB0`'s `secret_enc`
column to exist (even though `EB1` itself only reads `secret_hash` — sequenced first for a clean
migration history). `EB2` (charge/status/channel routes) is independent of `EB1` structurally but
needs it to be testable end-to-end (a route gated by `ActorApp` needs a way to obtain that
principal). `EB3` (outbound relay) needs `EB0`'s `secret_enc` directly. `EB4` (idempotency
behavior + rate limiting) touches `EB2`'s routes, sequenced after. `EB5` closes out with docs +
verification. Run `go build ./... && go vet ./... && go test ./...` after each phase.

#### Phase EB0 — Schema: `secret_enc` column + `JWT.AppTokenTTL` config

- [x] New migration: `ALTER TABLE payment_clients ADD COLUMN secret_enc VARBINARY(500) NULL
      AFTER secret_hash`. NULL for `kind=internal` rows, same convention as `secret_hash`.
- [x] `payment/infrastructure`'s `CreateApp`/`ResetAppSecret` (built in Part 2) gain one more
      write: encrypt the same plaintext secret (reusing the `encryptAESGCM` helper already built
      for gateway config) into `secret_enc` alongside the existing bcrypt hash. No change to
      either function's public behavior or return shape (§10.1.7 — no frontend change).
- [x] `config.JWT` gains `AppTokenTTL time.Duration` (default 1 hour, new env var
      `JWT_APP_TOKEN_TTL`, added to `.env`/`.env.example`).
- [x] `go build ./... && go vet ./...` clean.

#### Phase EB1 — `auth`: `ActorApp` + token issuance + live status check

Depends on `EB0`.

- [x] `authcontract.Actor` gains `ActorApp = "app"`.
- [x] `auth`'s own query file gains a narrow read: `id, app_id, secret_hash, status FROM
      payment_clients WHERE app_id = ? AND kind = 'external' LIMIT 1` (decision 2 — no contract
      call to `paymentclient.Client`).
- [x] New `POST /auth/app/token`: verify `secret` against `secret_hash` via the existing
      `security.VerifyPassword` (same helper every other login path already uses), reject
      inactive/not-found with the same generic `401` message every other login path already uses
      (don't leak whether an `app_id` exists — same information-disclosure discipline as human
      logins). On success, issue a JWT with `Actor: ActorApp`, `SubjectID: <payment_clients.id>`,
      TTL = `JWT.AppTokenTTL`, **no refresh token** (decision 3).
- [x] `auth/infrastructure/middleware.go`: for `Actor == ActorApp`, re-check `payment_clients
      .status == 'active'` on every request (decision 4) — reject `403` if inactive, same
      per-request-DB-read discipline as the existing §2.13/§2.15 checks (no caching).
- [x] Basic rate limiting on this endpoint keyed by source IP — 10 requests/minute (decision 11).
- [x] `go build ./... && go vet ./...` clean. Manual test: issue a token for the seeded (Part 2
      test-cleanup already removed it, so create a fresh) external app, confirm an inactive app
      is rejected, confirm deactivating an app mid-session 403s its very next authenticated call.
      **Verified live 2026-07-12** — see EB5's E2E note below.

#### Phase EB2 — `payment/presentation`: external charge/status/channel routes

Independent of `EB1` structurally; needs it to test end-to-end.

- [x] New routes in `payment/presentation` (decision 8): `POST /external/payments/charges`,
      `GET /external/payments/charges/{orderRef}/status`, `GET /external/payments/channels` — all
      gated `RequireActor(ActorApp)`.
- [x] Charge-creation handler resolves `appID` from `principal.SubjectID` via
      `GetPaymentClientByID` (already built, Part 2), then calls
      `CreateChannelCharge(ctx, appID, "", body.OrderRef, body.Amount, body.Channel,
      body.ChannelOptions)` (decision 5 — empty `storeID`).
- [x] Status-check handler: before calling `CheckStatus`, verify the resolved `orderRef` actually
      belongs to the calling `appID` (read `payment_charge_apps`) — reject **`404`, not `403`**,
      if it belongs to a DIFFERENT app (or doesn't exist at all — same response either way). An
      external caller must never be able to probe another app's charge status by guessing
      `orderRef` values; a `403` would itself leak "this orderRef exists, you're just not allowed
      to see it" — `404` gives no such signal. (This refines the `403` wording written when this
      bullet was first planned; implemented as `404` for that reason during `EB2`.)
- [x] `go build ./... && go vet ./...` clean.

#### Phase EB3 — Outbound webhook relay for `kind=external` apps

Depends on `EB0` (`secret_enc`).

- [x] `payment`'s `Dispatch` (§9.1.5) gains a branch: when the resolved `app_id`'s row has
      `kind='external'`, spawn a goroutine (decision 10) that decrypts `secret_enc`, builds the
      signed payload, and POSTs to `callback_url` with a short timeout (e.g. 10s) — `Dispatch`
      itself returns immediately after spawning it, without waiting.
- [x] `go build ./... && go vet ./...` clean. Manual test: register a test external app with
      `callback_url` pointing at a throwaway local HTTP listener, simulate a gateway webhook for
      one of its charges, confirm the listener receives a correctly-signed payload. **Verified
      live 2026-07-12** — see EB5's E2E note below.

#### Phase EB4 — Idempotency behavior + rate limiting on the charge/status routes

Depends on `EB2`.

- [x] Confirm (no new code expected — decision 9 says this is structural) that retrying
      `POST /external/payments/charges` with a previously-used `orderRef` returns `409` with a
      message pointing at the status-check endpoint, not a silent duplicate charge. **Verified
      live 2026-07-12.**
- [x] Rate limiting keyed by `app_id` on all three `/external/payments/*` routes — 60
      requests/minute (decision 11). **Verified live 2026-07-12** — 62 rapid calls yielded exactly
      60×`200` then 2×`429`.
- [x] `go build ./... && go vet ./...` clean.

#### Phase EB5 — Docs + final verification

Depends on `EB0`–`EB4`.

- [x] Written API reference at `docs/EXTERNAL_PAYMENT_API.md` (decision 12) — cover every route,
      the signature-verification algorithm for relayed webhooks, error codes, and the
      retry/idempotency contract.
- [x] Update `knowledge/MODULE_MAP.md` (`payment` row gains the external routes + `ActorApp`
      consumer), `knowledge/DATABASE_GUIDE.md` (`secret_enc` column), plus (beyond the original
      bullet) `docs/DB_SCHEMA.md` and `apps/api/db/README.md`'s migration tables, and
      `knowledge/DATABASE_GUIDE.md`'s migration-history list — same "every doc that lists
      migrations/columns" discipline §9.3/PB4 already established.
- [x] `go build ./... && go vet ./... && go test ./...` clean.
- [x] Manual smoke test, full loop: register an external app (existing Part 2 UI) → exchange
      secret for a token → create a charge with `storeID=""` → simulate a gateway webhook →
      confirm the relay arrives correctly signed at a local test receiver → poll the status
      endpoint and confirm it agrees with what the relay said → deactivate the app → confirm its
      still-unexpired token immediately stops working on all three routes.

**Verified 2026-07-12, real HTTP calls against the running dev API (`:8081`) — all green:**
logged in as superadmin, registered two fresh external test apps ("E2E Smoke Test App"
`E2E-SMOKE-TEST-APP-*`, callback pointed at a throwaway local HTTP listener on `:9911`).

- **Token issuance**: `POST /auth/app/token` with the fresh `appId`/secret → `200`, response had
  `accessToken` + `expiresIn: 3600`, confirmed **no** `refreshToken` field (decision 3).
- **Charge creation**: `POST /external/payments/charges` with `storeID` implicitly empty → `201`,
  real Tripay sandbox QRIS charge returned (`provider: "tripay"`, a real `providerRef`).
- **Idempotency**: retrying the exact same `orderRef` → `409 Conflict` pointing at the status
  endpoint, exactly per decision 9 — no silent duplicate.
- **Signed webhook relay**: POSTed a self-signed simulated Tripay callback (signed with the real
  `TRIPAY_PRIVATE_KEY` from `.env`, matching `tripay.go`'s own verification scheme) at
  `/webhooks/payment` → `200`; confirmed the relay arrived at the throwaway `:9911` listener with
  header `X-Elkasir-Signature: sha256=<hex>`; independently recomputed the HMAC using the app's
  own plaintext secret and confirmed it **matches exactly** — proves `secret_enc` round-trips
  correctly end-to-end (decision 6).
- **Status check**: polled `GET /external/payments/charges/{orderRef}/status` — correctly
  independent of the (self-simulated, not real-gateway) webhook, since it pulls live from Tripay's
  own sandbox merchant record rather than trusting the relay's payload — this is the intended
  design (§9.1.8's pull-based check is deliberately independent of the push path), not a defect.
- **Cross-app isolation**: a second registered app's token queried the first app's `orderRef` →
  `404`; queried a genuinely nonexistent `orderRef` → **the identical** `404` response body — con-
  firmed the two cases are indistinguishable, per the deliberate non-disclosure design (§10.2 EB2).
- **Deactivation revocation**: deactivated the first app via the existing "Aplikasi Terdaftar" UI
  path (`PATCH /platform/payment-clients/{id}/status`) — its still-unexpired token immediately got
  `403` on all three `/external/payments/*` routes on its very next call each, with zero delay
  (decision 4).
- **Rate limits**: 12 rapid `POST /auth/app/token` calls with a pinned `X-Forwarded-For` → first
  10 succeeded (as `401`s, since the credentials were intentionally invalid), 11th/12th got `429`
  — confirms the 10/min-per-IP limit (decision 11). 62 rapid `GET /external/payments/channels`
  calls under one app's token → exactly 60×`200` then 2×`429` — confirms the 60/min-per-`app_id`
  limit. (Note: testing the IP-keyed limit directly via `curl` without a pinned
  `X-Forwarded-For` under-counts, since each bare connection gets a fresh ephemeral port in
  `r.RemoteAddr` — chi's `RealIP` middleware only collapses this to a real client IP when a
  reverse proxy, e.g. production nginx, actually sets that header. Not a bug; a test-harness
  detail worth remembering next time this limiter is smoke-tested locally.)

Both test apps deactivated and left in the registry in `inactive` status (consistent with the
"never hard-deleted" convention — §9.1.3); the throwaway `:9911` listener process was stopped
after verification.

### 10.3 Explicit non-goals for Part 3 (don't build unless separately asked)

- Refund/void, payout/disbursement, or recurring/subscription-native billing via the external
  API — same exclusion as §9.4, for the same reasons; this part only exposes what §9.1.8 already
  built (channel-aware charge creation + status + listing), not new gateway capability.
- A refresh-token flow for `ActorApp` — decision 3; re-exchange via client-credentials instead.
- A persisted webhook-relay delivery log/audit table — decision 10; `slog` only, matching the
  email-notification precedent's lightness.
- Any new frontend UI — decision 7; the existing "Aplikasi Terdaftar" page (Part 2) needs no
  changes.
- Per-app custom rate limits, usage-based billing, or a usage-analytics dashboard for external
  callers — the flat, simple limiter in decision 11 is deliberately not tunable per app yet.
- CORS changes to allow browser-based external calls — decision 8 assumes server-to-server
  integration (the external app's own backend holds the secret); a browser-facing integration
  model is a different, unrequested design.

### 10.4 Open questions — all resolved as of 2026-07-12

All three original items here are now decided — kept below (struck through) for the same
traceability reason §9.5 keeps its resolved items, not deleted outright:

1. ~~`JWT_APP_TOKEN_TTL` default.~~ **RESOLVED — 1 hour, see §10.1.3/`EB0`.**
2. ~~Where the written API reference lives.~~ **RESOLVED — see §10.1.12: a dedicated
   `docs/EXTERNAL_PAYMENT_API.md`, not an `openapi.yaml` addition.**
3. ~~Rate limit thresholds.~~ **RESOLVED — see §10.1.11: 60 requests/minute per `app_id` on
   `/external/payments/*`, 10 requests/minute per source IP on `POST /auth/app/token`.**

Nothing currently blocks starting `EB0`.

### 10.5 If you're an agent resuming Part 3

**Part 3 is fully implemented and verified (§10.2's checkboxes + EB5's verification note)** —
there is no next phase to resume within it. If you're reading this because something in Part 3
regressed, EB5's verification note is the first place to check whether it's a known-and-fixed
issue resurfacing (it also records a test-harness gotcha with the IP-keyed rate limiter under bare
`curl`, worth re-reading before assuming a real regression). If you're picking up the **next**
initiative built on top of this API (a real external integrator, a delivery-log table, etc.):

1. Confirm Part 1 (§0–§8), Part 2 (§9.2/§9.3), and Part 3 (§10.2) are still fully green
   (`go build && go vet && go test`, `tsc --noEmit`) before starting — anything new here builds on
   all three.
2. Read §10.1 (locked decisions) in full first — especially decision 6 (the dual-storage secret
   design), which is the single most important piece of reasoning in this part and the easiest to
   accidentally simplify away if it's ever touched again.
3. §10.3 lists what was deliberately left out (refunds, a refresh-token flow, a delivery-log table,
   per-app rate-limit tuning, CORS/browser support) — read it before assuming any of those is
   simply "not built yet" by oversight; each was an explicit, reasoned exclusion.
4. If you add, remove, or reorder anything within §10, update every cross-reference to it within
   §10 in the same edit — same discipline §7/§9.8 already established for Parts 1 and 2.
