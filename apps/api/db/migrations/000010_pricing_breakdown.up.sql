-- Rincian biaya (service & gateway) untuk self_orders dan transactions.
-- self_orders sudah punya subtotal & total; transactions sudah punya tax.
ALTER TABLE self_orders
  ADD COLUMN service_charge BIGINT NOT NULL DEFAULT 0 AFTER subtotal,
  ADD COLUMN gateway_fee    BIGINT NOT NULL DEFAULT 0 AFTER service_charge,
  ADD COLUMN tax            BIGINT NOT NULL DEFAULT 0 AFTER gateway_fee;

ALTER TABLE transactions
  ADD COLUMN service_charge BIGINT NOT NULL DEFAULT 0 AFTER tax,
  ADD COLUMN gateway_fee    BIGINT NOT NULL DEFAULT 0 AFTER service_charge;
