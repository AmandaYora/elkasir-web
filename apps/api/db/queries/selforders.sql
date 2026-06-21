-- name: FindTableByCode :one
SELECT * FROM dining_tables WHERE code = ? LIMIT 1;

-- name: ListActiveProducts :many
SELECT p.id, p.name, COALESCE(c.name, '') AS category, p.price, COALESCE(p.image_url, '') AS image_url
FROM products p LEFT JOIN product_categories c ON c.id = p.category_id
WHERE p.store_id = ? AND p.status = 'active'
ORDER BY c.sort_order ASC, p.name ASC;

-- name: CreateSelfOrder :exec
INSERT INTO self_orders (
  id, store_id, table_id, status, payment_method, payment_status, claim_code,
  subtotal, service_charge, gateway_fee, tax, total, customer_note, expires_at
) VALUES (?, ?, ?, 'placed', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CreateSelfOrderItem :exec
INSERT INTO self_order_items (
  id, self_order_id, product_id, product_name, category, price, quantity, line_total, note
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetSelfOrder :one
SELECT * FROM self_orders WHERE id = ? AND store_id = ? LIMIT 1;

-- name: GetSelfOrderByID :one
SELECT * FROM self_orders WHERE id = ? LIMIT 1;

-- name: GetSelfOrderByClaimCode :one
SELECT * FROM self_orders WHERE store_id = ? AND claim_code = ? LIMIT 1;

-- name: ListSelfOrderItems :many
SELECT * FROM self_order_items WHERE self_order_id = ? ORDER BY created_at ASC;

-- name: UpdateSelfOrderStatus :execrows
UPDATE self_orders SET status = ? WHERE id = ? AND store_id = ?;

-- name: MarkSelfOrderPaid :exec
UPDATE self_orders SET payment_status = 'paid', transaction_id = ?, status = ? WHERE id = ?;

-- name: ExpireOverdueSelfOrders :execrows
UPDATE self_orders SET payment_status = 'expired'
WHERE store_id = ? AND payment_status = 'pending' AND expires_at IS NOT NULL AND expires_at < ?;
