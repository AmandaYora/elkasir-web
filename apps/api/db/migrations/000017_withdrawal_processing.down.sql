ALTER TABLE withdrawals
  DROP COLUMN rejected_reason,
  DROP COLUMN processed_at,
  DROP COLUMN claimed_at,
  DROP COLUMN processed_by;
