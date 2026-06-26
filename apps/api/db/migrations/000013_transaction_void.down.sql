ALTER TABLE transactions
  DROP COLUMN void_reason,
  DROP COLUMN voided_by,
  DROP COLUMN voided_at;
