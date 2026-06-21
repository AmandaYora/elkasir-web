-- Pengaturan pajak & layanan per toko (untuk breakdown harga & pemisahan keuangan).
-- PPN default OFF (admin meng-enable lewat menu Pengaturan); persen disimpan sebagai integer.
ALTER TABLE settings
  ADD COLUMN tax_enabled     TINYINT(1) NOT NULL DEFAULT 0  AFTER feature_qris,
  ADD COLUMN tax_percent     INT        NOT NULL DEFAULT 11 AFTER tax_enabled,
  ADD COLUMN service_percent INT        NOT NULL DEFAULT 2  AFTER tax_percent;
