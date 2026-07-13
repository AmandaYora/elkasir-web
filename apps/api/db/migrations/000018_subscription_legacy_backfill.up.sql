-- One-time backfill (PLAN.md §2.15/§6 item 1, Phase B1.5): before this feature, existing
-- tenants had NO store_subscriptions row at all — package-inactive is now an enforced access
-- gate, so pre-existing tenants are grandfathered onto a dedicated, real paid plan named
-- "Premium Contributor" — Rp1.700.000/tahun, `is_active = 0` (never shown in the tenant-facing
-- plan picker — only ever assignable via this migration, never selectable at checkout) and
-- `renewal_only = 1` (once on it, a tenant may only ever renew the SAME plan — `subscription`
-- module's Checkout() rejects switching away from OR into this plan by any other path; see
-- application/service.go). The first period is 365 days from migration time (not the original
-- design's 20-year grace) — a legacy tenant gets one full year on the house, then renews at the
-- real Rp1.700.000 price like a genuine Premium Contributor from then on. Decided 2026-07-13.
--
-- IDs here use RANDOM_BYTES(13) hex-encoded (26 chars, fits the CHAR(26) ULID column width) —
-- migrations have no access to the app's Go ULID generator, and this is a one-time, one-off
-- backfill, not an ongoing write path.

INSERT INTO subscription_plans (id, code, name, price, period_days, is_active, renewal_only, created_at, updated_at)
SELECT LOWER(HEX(RANDOM_BYTES(13))), 'premium-contributor', 'Premium Contributor', 1700000, 365, 0, 1, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM subscription_plans WHERE code = 'premium-contributor');

INSERT INTO store_subscriptions (id, store_id, plan_id, status, current_period_start, current_period_end, created_at, updated_at)
SELECT LOWER(HEX(RANDOM_BYTES(13))), s.id,
       (SELECT id FROM subscription_plans WHERE code = 'premium-contributor' LIMIT 1),
       'active', NOW(), DATE_ADD(NOW(), INTERVAL 365 DAY), NOW(), NOW()
FROM stores s
WHERE NOT EXISTS (SELECT 1 FROM store_subscriptions ss WHERE ss.store_id = s.id);
