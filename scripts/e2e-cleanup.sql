-- Cleanup SEMUA data hasil E2E testing. Mengembalikan elkasir_db ke baseline seed:
-- store "Elkasir" + admin "admin" + settings-nya. Menghapus fixture uji, store B (tenancy),
-- admin tambahan, dan seluruh data transaksional yang dihasilkan pengujian.
SET FOREIGN_KEY_CHECKS = 0;

-- Data transaksional (semua dihasilkan oleh pengujian).
DELETE FROM self_order_items;
DELETE FROM payments;
DELETE FROM self_orders;
DELETE FROM transaction_items;
DELETE FROM transactions;
DELETE FROM cash_movements;
DELETE FROM shifts;
DELETE FROM idempotency_keys;
DELETE FROM webhook_events;
DELETE FROM refresh_tokens;
DELETE FROM withdrawals;

-- Fixture uji (dibuat khusus untuk pengujian).
DELETE FROM staff;
DELETE FROM products;
DELETE FROM product_categories;
DELETE FROM dining_tables;

-- Multi-tenant fixture: hapus store B + admin tambahan; sisakan baseline 'Elkasir' / 'admin'.
DELETE FROM stores       WHERE name <> 'Elkasir';
DELETE FROM admin_users  WHERE email <> 'admin';
DELETE FROM settings     WHERE store_id NOT IN (SELECT id FROM stores);

SET FOREIGN_KEY_CHECKS = 1;
