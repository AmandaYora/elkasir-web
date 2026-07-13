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
INSERT INTO subscription_invoices (id, store_id, plan_id, amount, status, provider, provider_ref)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetSubscriptionInvoice :one
SELECT * FROM subscription_invoices WHERE id = ? AND store_id = ? LIMIT 1;

-- name: GetSubscriptionInvoiceByID :one
SELECT * FROM subscription_invoices WHERE id = ? LIMIT 1;

-- name: MarkSubscriptionInvoicePaid :execrows
UPDATE subscription_invoices SET status = 'paid', period_start = ?, period_end = ?
WHERE id = ? AND status = 'pending';

-- name: ListSubscriptionInvoices :many
SELECT * FROM subscription_invoices WHERE store_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountSubscriptionInvoices :one
SELECT COUNT(*) FROM subscription_invoices WHERE store_id = ?;
