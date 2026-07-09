-- Logo toko (URL object storage, diunggah lewat POST /uploads?category=store-logo).
-- Ditampilkan di header struk mobile bersama name/address/phone yang sudah ada.
ALTER TABLE stores
  ADD COLUMN logo_url VARCHAR(500) NULL AFTER phone;
