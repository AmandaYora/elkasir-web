-- Tabel settings dimiliki modul `settings`. Pembaca lintas-modul memakai kontrak
-- settingsclient; query di sini adalah satu-satunya akses tulis.

-- name: GetSettingsByStore :one
SELECT * FROM settings WHERE store_id = ? LIMIT 1;

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
