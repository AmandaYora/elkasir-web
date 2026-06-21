-- name: CreatePayment :exec
INSERT INTO payments (id, store_id, self_order_id, provider, provider_ref, method, amount, status, raw_payload)
VALUES (?, ?, ?, ?, ?, 'qris', ?, ?, ?);

-- name: GetPaymentByProviderRef :one
SELECT * FROM payments WHERE provider_ref = ? LIMIT 1;

-- name: GetPaymentBySelfOrder :one
SELECT * FROM payments WHERE self_order_id = ? ORDER BY created_at DESC LIMIT 1;

-- name: UpdatePaymentStatus :exec
UPDATE payments SET status = ?, raw_payload = ? WHERE id = ?;

-- name: GetWebhookEvent :one
SELECT * FROM webhook_events WHERE provider = ? AND event_id = ? LIMIT 1;

-- name: CreateWebhookEvent :exec
INSERT INTO webhook_events (id, provider, event_id) VALUES (?, ?, ?);
