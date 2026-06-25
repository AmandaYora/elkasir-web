-- Toggle metode "bayar di kasir" (tunai) untuk self-order, terpisah dari QRIS.
-- Admin bisa mematikan tiap metode independen; minimal satu wajib aktif saat self-order
-- aktif (divalidasi di service). Default ON agar perilaku toko lama tidak berubah.
ALTER TABLE settings
  ADD COLUMN feature_pay_at_cashier TINYINT(1) NOT NULL DEFAULT 1 AFTER feature_qris;
