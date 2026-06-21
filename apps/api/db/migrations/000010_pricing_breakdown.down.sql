ALTER TABLE transactions
  DROP COLUMN gateway_fee,
  DROP COLUMN service_charge;

ALTER TABLE self_orders
  DROP COLUMN tax,
  DROP COLUMN gateway_fee,
  DROP COLUMN service_charge;
