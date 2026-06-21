-- Kembalikan enum ke set sebelum Tripay. Jalankan hanya bila tak ada baris ber-provider
-- 'tripay' (akan menjadi string kosong saat downgrade).
ALTER TABLE payments
  MODIFY provider ENUM('xendit','midtrans') NOT NULL DEFAULT 'midtrans';
