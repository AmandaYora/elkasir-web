-- Fase 1 — "Bebas dari Penjara Foreign Key".
-- Drop 16 FK BISNIS LINTAS-MODUL. Kolom & index DIPERTAHANKAN (MySQL tidak menghapus
-- index saat DROP FOREIGN KEY) sehingga performa join/filter tetap. Integritas relasi
-- lintas-modul kini ditegakkan di logic layer (module client), bukan constraint DB.
--
-- TIDAK di-drop (sengaja): FK intra-modul (fk_products_category, fk_transaction_items_tx,
-- fk_self_order_items_order) + semua FK `store_id -> stores` (kunci tenant, shared-kernel).
-- Lihat docs/architecture/modular-monolith.md §3.1.

-- shift -> staff
ALTER TABLE shifts DROP FOREIGN KEY fk_shifts_staff;
ALTER TABLE shifts DROP FOREIGN KEY fk_shifts_approver;

-- transaction -> shift / table / staff
ALTER TABLE transactions DROP FOREIGN KEY fk_transactions_shift;
ALTER TABLE transactions DROP FOREIGN KEY fk_transactions_table;
ALTER TABLE transactions DROP FOREIGN KEY fk_transactions_cashier;
ALTER TABLE transactions DROP FOREIGN KEY fk_transactions_discount_approver;

-- transaction(item) -> product
ALTER TABLE transaction_items DROP FOREIGN KEY fk_transaction_items_product;

-- cashmovement -> shift / staff
ALTER TABLE cash_movements DROP FOREIGN KEY fk_cash_movements_shift;
ALTER TABLE cash_movements DROP FOREIGN KEY fk_cash_movements_creator;
ALTER TABLE cash_movements DROP FOREIGN KEY fk_cash_movements_approver;

-- withdrawal -> adminuser
ALTER TABLE withdrawals DROP FOREIGN KEY fk_withdrawals_requester;

-- selforder -> table / product
ALTER TABLE self_orders DROP FOREIGN KEY fk_self_orders_table;
ALTER TABLE self_order_items DROP FOREIGN KEY fk_self_order_items_product;

-- payment -> selforder
ALTER TABLE payments DROP FOREIGN KEY fk_payments_self_order;

-- tautan silang transaction <-> selforder
ALTER TABLE transactions DROP FOREIGN KEY fk_transactions_self_order;
ALTER TABLE self_orders DROP FOREIGN KEY fk_self_orders_transaction;
