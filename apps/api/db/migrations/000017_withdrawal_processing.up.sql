-- Withdrawal claim -> complete flow (PLAN.md §2.7/§2.8, Phase B0). Semua kolom baru adalah
-- ID primitif / metadata proses saja — tanpa FK lintas modul (processed_by -> platform_users,
-- sesuai konvensi "Bebas dari Penjara FK" yang sama dipakai requested_by di migration 000005).
ALTER TABLE withdrawals
  ADD COLUMN processed_by    CHAR(26)     NULL AFTER requested_by,
  ADD COLUMN claimed_at      DATETIME     NULL AFTER processed_by,
  ADD COLUMN processed_at    DATETIME     NULL AFTER claimed_at,
  ADD COLUMN rejected_reason VARCHAR(255) NULL AFTER processed_at;
