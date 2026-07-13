-- Tabel settings dimiliki modul `settings`. Pembaca lintas-modul memakai kontrak
-- settingsclient; query di sini adalah satu-satunya akses tulis.

-- name: GetSettingsByStore :one
SELECT * FROM settings WHERE store_id = ? LIMIT 1;

-- stores adalah shared kernel; modul settings juga mengelola kolom profil (name/address/
-- phone/logo_url) sebagai bagian dari "menu Pengaturan" — lihat knowledge/MODULE_MAP.md.
-- Kolom lain di stores (type/timezone/currency) TIDAK disentuh dari sini.

-- slug dibaca (read-only) di sini untuk ditampilkan admin (URL self-order publik
-- /order/<slug>/<kodeMeja>) — kepemilikan TULIS-nya tetap di modul `platform` (lihat
-- migration 000016 & knowledge/MODULE_MAP.md); settings tidak pernah menulis slug.
-- name: GetStoreProfile :one
SELECT id, name, slug, address, phone, logo_url FROM stores WHERE id = ? LIMIT 1;

-- name: UpdateStoreProfile :exec
UPDATE stores
SET name = ?, address = ?, phone = ?, logo_url = ?
WHERE id = ?;

-- name: UpsertSettings :exec
INSERT INTO settings (
  id, store_id, max_discount_percent, max_operational_expense, cash_variance_tolerance,
  feature_self_order, feature_qris, feature_pay_at_cashier, tax_enabled, tax_percent, service_percent
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  max_discount_percent    = VALUES(max_discount_percent),
  max_operational_expense = VALUES(max_operational_expense),
  cash_variance_tolerance = VALUES(cash_variance_tolerance),
  feature_self_order      = VALUES(feature_self_order),
  feature_qris            = VALUES(feature_qris),
  feature_pay_at_cashier  = VALUES(feature_pay_at_cashier),
  tax_enabled             = VALUES(tax_enabled),
  tax_percent             = VALUES(tax_percent),
  service_percent         = VALUES(service_percent);
