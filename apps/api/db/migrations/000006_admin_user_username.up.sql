-- Tambah kolom `username` untuk pengguna admin (dashboard web).
-- Pengguna admin kini dapat login dengan username ATAU email (lihat auth.LoginAdmin).
-- Kolom NULLABLE + UNIQUE: baris admin lama (mis. hasil seed) boleh tanpa username
-- (MySQL mengizinkan banyak NULL pada index UNIQUE), tetapi pengguna baru wajib unik.
-- Aditif & backward-compatible (image lama tetap jalan setelah migrasi ini).
ALTER TABLE admin_users ADD COLUMN username VARCHAR(100) NULL AFTER email;
ALTER TABLE admin_users ADD UNIQUE KEY uq_admin_users_username (username);
