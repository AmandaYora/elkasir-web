-- Lepas FK silang dulu agar tabel bisa di-drop.
ALTER TABLE self_orders DROP FOREIGN KEY fk_self_orders_transaction;
ALTER TABLE transactions DROP FOREIGN KEY fk_transactions_self_order;

DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS self_order_items;
DROP TABLE IF EXISTS self_orders;
