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
-- ElProof (elproof_*) is a SEPARATE, always-on wallet used only for subscription billing
-- (paymentclient.AppSubscribe) — not part of the Provider switch above (§11).
UPDATE payment_gateway_config SET
  provider = ?, sandbox = ?, tripay_api_key_enc = ?, tripay_private_key_enc = ?,
  tripay_merchant_code_enc = ?, tripay_method = ?, midtrans_server_key_enc = ?,
  elproof_app_id = ?, elproof_secret_enc = ?, elproof_base_url = ?
WHERE id = ?;

-- name: GetChargeApp :one
SELECT app_id FROM payment_charge_apps WHERE order_ref = ? LIMIT 1;

-- name: CreateChargeApp :exec
INSERT INTO payment_charge_apps (order_ref, app_id) VALUES (?, ?);
