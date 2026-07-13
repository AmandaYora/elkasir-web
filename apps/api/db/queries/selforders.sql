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

-- Ledger pembayaran gateway (tabel `payments`) — dimiliki selforder (satu-satunya pemakai);
-- module `payment` sendiri sudah tidak menyentuh tabel ini (lihat webhook_events.sql).

-- name: CreatePayment :exec
INSERT INTO payments (id, store_id, self_order_id, provider, provider_ref, method, amount, status, raw_payload)
VALUES (?, ?, ?, ?, ?, 'qris', ?, ?, ?);

-- name: GetPaymentBySelfOrder :one
SELECT * FROM payments WHERE self_order_id = ? ORDER BY created_at DESC LIMIT 1;

-- name: UpdatePaymentStatus :exec
UPDATE payments SET status = ?, raw_payload = ? WHERE id = ?;
