-- Ganti gateway pembayaran ke Midtrans. Expand-only (backward compatible): 'xendit' tetap
-- diterima agar baris lama valid & image lama tetap jalan setelah rollback; default baru
-- 'midtrans'.
ALTER TABLE payments
  MODIFY provider ENUM('xendit','midtrans') NOT NULL DEFAULT 'midtrans';
