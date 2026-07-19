ALTER TABLE subscription_invoices
  DROP INDEX uq_subscription_invoices_pending_lock,
  DROP COLUMN pending_lock_key,
  DROP COLUMN store_id_shadow;
