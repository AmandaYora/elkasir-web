-- Platform layer: superadmin identity + tenant lifecycle, and 2 multi-tenancy fixes found
-- during audit (kode meja & username staff cuma unik per-toko, padahal entry point-nya
-- belum tahu toko mana sebelum lookup).

-- Identitas superadmin (operator platform) — SENGAJA tanpa store_id sama sekali; ini satu-
-- satunya identitas di seluruh skema yang tidak terikat tenant mana pun.
CREATE TABLE platform_users (
  id             CHAR(26)     NOT NULL,
  name           VARCHAR(150) NOT NULL,
  email          VARCHAR(190) NOT NULL,
  password_hash  VARCHAR(100) NOT NULL,
  status         ENUM('active','inactive') NOT NULL DEFAULT 'active',
  created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_platform_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Siklus hidup tenant — kolom dimiliki modul `platform` (pengecualian shared-kernel pada
-- `stores`, pola yang sama dengan kolom profil milik `settings` — lihat MODULE_MAP.md).
ALTER TABLE stores ADD COLUMN slug VARCHAR(60) NULL AFTER name;
ALTER TABLE stores ADD COLUMN status ENUM('active','suspended') NOT NULL DEFAULT 'active' AFTER slug;

-- Backfill slug utk baris lama (toko hasil bootstrap seed) SEBELUM constraint unik ditegakkan.
UPDATE stores SET slug = LOWER(REPLACE(REPLACE(REPLACE(name, ' ', '-'), '.', ''), '_', '-'))
WHERE slug IS NULL;
ALTER TABLE stores MODIFY COLUMN slug VARCHAR(60) NOT NULL;
ALTER TABLE stores ADD UNIQUE KEY uq_stores_slug (slug);

-- Fix bug: staff.username hanya unik (store_id, username) — tapi endpoint login staff
-- (dipakai APK POS yang SAMA di semua tenant) tidak membawa identitas toko sama sekali, jadi
-- lookup-nya (`GetStaffByUsername`) tidak bisa (dan tidak boleh) di-scope. Solusi paling aman
-- (nol dampak ke app mobile — kontrak request login tetap {username,password}): username
-- dibuat unik GLOBAL, sama seperti admin_users.username (migration 000006).
ALTER TABLE staff DROP INDEX uq_staff_store_username;
ALTER TABLE staff ADD UNIQUE KEY uq_staff_username (username);

-- refresh_tokens perlu mendukung sesi actor 'platform' juga (expand-only, backward compatible).
ALTER TABLE refresh_tokens MODIFY COLUMN actor ENUM('admin','staff','platform') NOT NULL;
