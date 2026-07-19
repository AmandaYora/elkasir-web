-- Reconciliation poller (subscription's ReconcilePending, ticks every 2 min, PLAN.md §11 Part C)
-- filters `WHERE status = 'pending' AND provider = 'elproof'` on every tick with no covering
-- index (only store_id/provider_ref were indexed) — this becomes a full table scan as
-- subscription_invoices grows, forever, since the table is never archived. This composite index
-- makes that exact filter (plus the ORDER BY created_at ASC LIMIT ? it's paired with) a direct
-- index range scan instead.
ALTER TABLE subscription_invoices
  ADD INDEX idx_subscription_invoices_status_provider_created (status, provider, created_at);
