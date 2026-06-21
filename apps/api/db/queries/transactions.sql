-- name: CreateTransaction :exec
INSERT INTO transactions (
  id, store_id, code, shift_id, table_id, self_order_id, cashier_id, order_type, source,
  payment_method, status, subtotal, discount, tax, total, amount_received, change_amount,
  discount_approved_by, customer_note, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CreateTransactionItem :exec
INSERT INTO transaction_items (
  id, transaction_id, product_id, product_name, category, price, quantity, line_total, note
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetProductForSale :one
SELECT p.id, p.name, COALESCE(c.name, '') AS category, p.price, p.status, p.stock
FROM products p LEFT JOIN product_categories c ON c.id = p.category_id
WHERE p.id = ? AND p.store_id = ? LIMIT 1;

-- name: GetTransaction :one
SELECT * FROM transactions WHERE id = ? AND store_id = ? LIMIT 1;

-- name: ListTransactionItems :many
SELECT * FROM transaction_items WHERE transaction_id = ? ORDER BY created_at ASC;

-- Kurangi stok dengan penjaga stok >= qty (cegah stok negatif). 0 rows = gagal.
-- name: DecrementStock :execrows
UPDATE products SET stock = stock - sqlc.arg(qty)
WHERE id = sqlc.arg(id) AND store_id = sqlc.arg(store_id) AND stock >= sqlc.arg(qty);

-- name: GetIdempotencyKey :one
SELECT * FROM idempotency_keys WHERE store_id = ? AND idempotency_key = ? LIMIT 1;

-- name: CreateIdempotencyKey :exec
INSERT INTO idempotency_keys (id, store_id, idempotency_key, request_hash, response_status, response_body)
VALUES (?, ?, ?, ?, ?, ?);
