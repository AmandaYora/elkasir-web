-- name: ListPaymentClients :many
SELECT * FROM payment_clients ORDER BY created_at ASC;

-- name: GetPaymentClientByAppID :one
SELECT * FROM payment_clients WHERE app_id = ? LIMIT 1;

-- name: GetPaymentClientByID :one
SELECT * FROM payment_clients WHERE id = ? LIMIT 1;

-- name: CreatePaymentClient :exec
INSERT INTO payment_clients (id, app_id, name, secret_hash, secret_enc, kind, callback_url, status)
VALUES (?, ?, ?, ?, ?, ?, ?, 'active');

-- name: SetPaymentClientSecret :execrows
UPDATE payment_clients SET secret_hash = ?, secret_enc = ? WHERE id = ? AND kind = 'external';

-- name: GetPaymentClientSecretEnc :one
-- Dipakai HANYA saat menandatangani relay webhook keluar (§10.1.6/§10.1.10) — bukan untuk
-- otentikasi masuk (itu pakai secret_hash via GetPaymentClientByAppID/ByID + bcrypt compare).
SELECT secret_enc FROM payment_clients WHERE id = ? AND kind = 'external' LIMIT 1;

-- name: SetPaymentClientStatus :execrows
UPDATE payment_clients SET status = ? WHERE id = ? AND kind = 'external';

-- name: GetPaymentGatewayConfig :one
SELECT * FROM payment_gateway_config LIMIT 1;

-- name: CountPaymentGatewayConfig :one
SELECT COUNT(*) FROM payment_gateway_config;

-- name: InsertPaymentGatewayConfig :exec
INSERT INTO payment_gateway_config (
  id, provider, sandbox, tripay_api_key_enc, tripay_private_key_enc, tripay_merchant_code_enc,
  tripay_method, midtrans_server_key_enc
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePaymentGatewayConfig :exec
UPDATE payment_gateway_config SET
  provider = ?, sandbox = ?, tripay_api_key_enc = ?, tripay_private_key_enc = ?,
  tripay_merchant_code_enc = ?, tripay_method = ?, midtrans_server_key_enc = ?
WHERE id = ?;

-- name: GetChargeApp :one
SELECT app_id FROM payment_charge_apps WHERE order_ref = ? LIMIT 1;

-- name: GetChargeAppByOrderRef :one
-- Termasuk provider_ref (§10.2 EB2) — dipakai endpoint status eksternal untuk menerjemahkan
-- orderRef milik pemanggil ke providerRef yang CheckStatus benar-benar butuhkan.
SELECT app_id, provider_ref FROM payment_charge_apps WHERE order_ref = ? LIMIT 1;

-- name: CreateChargeApp :exec
INSERT INTO payment_charge_apps (order_ref, app_id, provider_ref) VALUES (?, ?, ?);
