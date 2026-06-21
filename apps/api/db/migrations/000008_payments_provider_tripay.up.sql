-- Dukung gateway pembayaran multi-provider. Expand-only (backward compatible): tambah
-- 'tripay' ke enum; 'xendit'/'midtrans' tetap valid. Default disetel ke 'tripay' (provider
-- aktif saat ini); provider sebenarnya tetap diisi eksplisit oleh aplikasi per-baris.
ALTER TABLE payments
  MODIFY provider ENUM('xendit','midtrans','tripay') NOT NULL DEFAULT 'tripay';
