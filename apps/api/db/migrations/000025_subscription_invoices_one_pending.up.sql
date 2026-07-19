-- Enforces "at most one pending invoice per store" as a DB-level invariant, not just an
-- application-level check-then-insert (which has an inherent TOCTOU race under concurrent
-- checkout requests — see subscription/application/service.go's Checkout: a fast pre-check gives
-- a friendly error in the common case, but only a real DB constraint closes the race for two
-- checkout requests landing at the same instant). MySQL has no native partial/filtered unique
-- index, so this uses the standard workaround: a generated column that's NULL unless
-- status = 'pending' (MySQL's unique index permits any number of NULL entries), with a unique
-- index on it.
--
-- The generated column is derived from `store_id_shadow`, NOT `store_id` directly, even though
-- they always hold the same value: InnoDB refuses to add a STORED generated column whose
-- expression reads a column that is the child side of a foreign key with an ON DELETE CASCADE
-- action (verified empirically against this exact table/FK — MySQL 8.0.30 fails with
-- "Error 1215: Cannot add foreign key constraint" otherwise). `store_id_shadow` is a plain,
-- ordinary column with no FK of its own, set to the same value as `store_id` at insert time
-- (see db/queries/subscriptions.sql's CreateSubscriptionInvoice) purely so the generated column
-- has a legal, non-FK'd base column to read — it is never queried directly and is deleted along
-- with its row exactly like every other column when `fk_subscription_invoices_store`'s
-- ON DELETE CASCADE fires (that FK itself is untouched by this migration).
ALTER TABLE subscription_invoices
  ADD COLUMN store_id_shadow CHAR(26) NULL,
  ADD COLUMN pending_lock_key CHAR(26)
    GENERATED ALWAYS AS (CASE WHEN status = 'pending' THEN store_id_shadow ELSE NULL END) STORED,
  ADD UNIQUE INDEX uq_subscription_invoices_pending_lock (pending_lock_key);
