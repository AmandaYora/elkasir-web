-- PLAN.md §11: Elkasir stops being a payment-gateway-as-a-service PROVIDER (Part 3, PLAN.md §10)
-- -- it becomes a CLIENT of a separate product (ElProof) instead, only for subscription billing.
-- Removes the columns/rows that only ever existed to serve external (kind='external') callers of
-- Elkasir's own Tripay/Midtrans wallet. `kind`/`callback_url`/`secret_hash` columns are left in
-- place on payment_clients (harmless residual structure -- not worth an enum-shrink migration).
DELETE FROM payment_clients WHERE kind = 'external';
ALTER TABLE payment_clients DROP COLUMN secret_enc;
ALTER TABLE payment_charge_apps DROP COLUMN provider_ref;
