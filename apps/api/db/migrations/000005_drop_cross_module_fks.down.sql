-- Reversal Fase 1 — pasang kembali ke-16 FK bisnis lintas-modul (definisi identik migrasi
-- 000003 & 000004). Mengasumsikan data konsisten (tak ada id yatim) saat di-apply.

-- shift -> staff
ALTER TABLE shifts ADD CONSTRAINT fk_shifts_staff FOREIGN KEY (staff_id) REFERENCES staff (id);
ALTER TABLE shifts ADD CONSTRAINT fk_shifts_approver FOREIGN KEY (close_approved_by) REFERENCES staff (id);

-- transaction -> shift / table / staff
ALTER TABLE transactions ADD CONSTRAINT fk_transactions_shift FOREIGN KEY (shift_id) REFERENCES shifts (id);
ALTER TABLE transactions ADD CONSTRAINT fk_transactions_table FOREIGN KEY (table_id) REFERENCES dining_tables (id);
ALTER TABLE transactions ADD CONSTRAINT fk_transactions_cashier FOREIGN KEY (cashier_id) REFERENCES staff (id);
ALTER TABLE transactions ADD CONSTRAINT fk_transactions_discount_approver FOREIGN KEY (discount_approved_by) REFERENCES staff (id);

-- transaction(item) -> product
ALTER TABLE transaction_items ADD CONSTRAINT fk_transaction_items_product FOREIGN KEY (product_id) REFERENCES products (id) ON DELETE SET NULL;

-- cashmovement -> shift / staff
ALTER TABLE cash_movements ADD CONSTRAINT fk_cash_movements_shift FOREIGN KEY (shift_id) REFERENCES shifts (id);
ALTER TABLE cash_movements ADD CONSTRAINT fk_cash_movements_creator FOREIGN KEY (created_by) REFERENCES staff (id);
ALTER TABLE cash_movements ADD CONSTRAINT fk_cash_movements_approver FOREIGN KEY (approved_by) REFERENCES staff (id);

-- withdrawal -> adminuser
ALTER TABLE withdrawals ADD CONSTRAINT fk_withdrawals_requester FOREIGN KEY (requested_by) REFERENCES admin_users (id);

-- selforder -> table / product
ALTER TABLE self_orders ADD CONSTRAINT fk_self_orders_table FOREIGN KEY (table_id) REFERENCES dining_tables (id);
ALTER TABLE self_order_items ADD CONSTRAINT fk_self_order_items_product FOREIGN KEY (product_id) REFERENCES products (id) ON DELETE SET NULL;

-- payment -> selforder
ALTER TABLE payments ADD CONSTRAINT fk_payments_self_order FOREIGN KEY (self_order_id) REFERENCES self_orders (id) ON DELETE CASCADE;

-- tautan silang transaction <-> selforder
ALTER TABLE transactions ADD CONSTRAINT fk_transactions_self_order FOREIGN KEY (self_order_id) REFERENCES self_orders (id);
ALTER TABLE self_orders ADD CONSTRAINT fk_self_orders_transaction FOREIGN KEY (transaction_id) REFERENCES transactions (id);
