-- Void (pembatalan transaksi tunai dalam shift berjalan). Enum status sudah memuat 'voided';
-- migrasi ini hanya MENAMBAH kolom jejak audit (additive/expand — aman untuk rollback).
ALTER TABLE transactions
  ADD COLUMN voided_at   DATETIME     NULL AFTER status,
  ADD COLUMN voided_by   CHAR(26)     NULL AFTER voided_at,
  ADD COLUMN void_reason VARCHAR(255) NULL AFTER voided_by;
