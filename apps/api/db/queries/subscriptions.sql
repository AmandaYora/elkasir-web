-- name: ListActiveSubscriptionPlans :many
SELECT * FROM subscription_plans WHERE is_active = 1 ORDER BY price ASC;

-- name: ListAllSubscriptionPlans :many
-- Dipakai platform (superadmin) — termasuk plan nonaktif, tidak seperti ListActiveSubscriptionPlans.
SELECT * FROM subscription_plans ORDER BY price ASC;

-- name: GetSubscriptionPlan :one
SELECT * FROM subscription_plans WHERE id = ? LIMIT 1;

-- name: CreateSubscriptionPlan :exec
INSERT INTO subscription_plans (id, code, name, price, period_days, is_active)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateSubscriptionPlan :execrows
UPDATE subscription_plans SET name = ?, price = ?, period_days = ?, is_active = ? WHERE id = ?;

-- name: SumPaidSubscriptionInvoices :one
-- Revenue platform (superadmin): total seluruh invoice LUNAS, LINTAS SEMUA TENANT — sengaja
-- tanpa filter store_id, satu-satunya query di modul ini yang boleh begitu.
SELECT CAST(COALESCE(SUM(amount), 0) AS SIGNED) FROM subscription_invoices WHERE status = 'paid';

-- name: GetStoreSubscription :one
SELECT * FROM store_subscriptions WHERE store_id = ? LIMIT 1;

-- name: UpsertStoreSubscriptionPeriod :exec
INSERT INTO store_subscriptions (id, store_id, plan_id, status, current_period_start, current_period_end)
VALUES (?, ?, ?, 'active', ?, ?)
ON DUPLICATE KEY UPDATE
  plan_id = VALUES(plan_id),
  status = 'active',
  current_period_start = VALUES(current_period_start),
  current_period_end = VALUES(current_period_end);

-- name: CreateSubscriptionInvoice :exec
-- store_id_shadow is always set equal to store_id (see migration 000025's doc comment for why a
-- separate column exists at all — an InnoDB restriction on generated columns + ON DELETE CASCADE
-- FKs, not a real second piece of data). A duplicate-key error on this insert (MySQL error 1062)
-- means the store already has a pending invoice open — surfaced by the repository as
-- domain.ErrInvoiceAlreadyPending.
INSERT INTO subscription_invoices (id, store_id, plan_id, amount, status, provider, provider_ref, store_id_shadow)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetSubscriptionInvoice :one
SELECT * FROM subscription_invoices WHERE id = ? AND store_id = ? LIMIT 1;

-- name: GetSubscriptionInvoiceByID :one
SELECT * FROM subscription_invoices WHERE id = ? LIMIT 1;

-- name: GetPendingSubscriptionInvoiceByStore :one
-- Backs the checkout double-submit guard: a store may only have ONE unresolved invoice open
-- at a time, regardless of provider. Most-recent first in case more than one somehow exists
-- (shouldn't, given this guard, but pre-existing data might).
SELECT * FROM subscription_invoices WHERE store_id = ? AND status = 'pending' ORDER BY created_at DESC LIMIT 1;

-- name: SetSubscriptionInvoiceProviderRef :exec
-- Filled in AFTER a successful gateway charge (see subscription/application/service.go Checkout)
-- — informational only for ElProof invoices (status checks key off the invoice's own ID as
-- orderRef, never providerRef), kept for ops/support traceability against ElProof's own charge
-- log. A failure to persist this is logged but never fails checkout.
UPDATE subscription_invoices SET provider_ref = ? WHERE id = ?;

-- name: MarkSubscriptionInvoicePaid :execrows
UPDATE subscription_invoices SET status = 'paid', period_start = ?, period_end = ?
WHERE id = ? AND status = 'pending';

-- name: MarkSubscriptionInvoiceTerminal :execrows
-- Closes out an invoice that ElProof reports as genuinely done WITHOUT being paid (expired or
-- failed/refund — subscription_invoices has no separate 'refund' state, so refund maps to
-- 'failed', the closest fit) — guarded by status='pending' so a late webhook/reconciler tick
-- can never downgrade an invoice that already resolved to 'paid'.
UPDATE subscription_invoices SET status = ? WHERE id = ? AND status = 'pending';

-- name: ListSubscriptionInvoices :many
SELECT * FROM subscription_invoices WHERE store_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: ListPendingElProofInvoices :many
-- Reconciliation poller fallback (PLAN.md §11 Part C) — ElProof's webhook relay is best-effort,
-- single-attempt; unlike the old in-process dispatch, a lost relay is now a real possibility, so
-- this backs GET status-check polling for anything still pending. LIMIT bounds each tick to a
-- fixed-size batch (mirrors ElProof's own reconcileBatchLimit=50 on its sweep) so a large backlog
-- can't turn one tick into an unbounded burst of outbound requests to ElProof.
SELECT * FROM subscription_invoices WHERE status = 'pending' AND provider = 'elproof' ORDER BY created_at ASC LIMIT ?;

-- name: CountSubscriptionInvoices :one
SELECT COUNT(*) FROM subscription_invoices WHERE store_id = ?;
