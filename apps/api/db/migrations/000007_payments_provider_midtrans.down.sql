-- Kembalikan provider ke ENUM lama. Catatan: baris ber-provider 'midtrans' akan menjadi
-- string kosong bila ada saat downgrade; jalankan hanya bila tak ada data Midtrans.
ALTER TABLE payments
  MODIFY provider ENUM('xendit') NOT NULL DEFAULT 'xendit';
