-- PIN supervisor untuk persetujuan in-place (approve-in-place) di POS — supervisor mengetik
-- PIN singkat untuk mengotorisasi aksi kasir (diskon/varians di atas ambang). Hanya
-- supervisor yang mengisi; disimpan ter-hash (bcrypt). NULL = belum diset.
ALTER TABLE staff
  ADD COLUMN pin_hash VARCHAR(100) NULL AFTER password_hash;
