-- Part 3 (PLAN.md §10, EB2) — gap found during implementation: the external status-check
-- endpoint (GET /external/payments/charges/{orderRef}/status) only ever knows the CALLER's own
-- `orderRef` (what they supplied at charge-creation time), never the gateway's own
-- `provider_ref` that CheckStatus actually needs — and `payment` deliberately owns no business
-- ledger to look that mapping up in (§9's core principle). Extending the existing thin dispatch
-- index (`payment_charge_apps`, already NOT a ledger — just order_ref/app_id routing metadata)
-- with one more column is a minor, well-justified extension of its existing job, not a new
-- ledger: it's still populated at the exact same write (CreateChannelCharge), just one column
-- wider.
ALTER TABLE payment_charge_apps ADD COLUMN provider_ref VARCHAR(191) NULL AFTER app_id;
