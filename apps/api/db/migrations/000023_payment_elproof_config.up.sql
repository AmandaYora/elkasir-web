-- ElProof integration (PLAN.md §11) -- a SEPARATE, always-on wallet used only for subscription
-- billing (paymentclient.AppSubscribe), living alongside (not replacing) the existing
-- Provider-selected Tripay/Midtrans wallet used by selforder. Same encrypt-at-rest convention as
-- the Tripay/Midtrans columns already in this table (AES-256-GCM, CONFIG_ENCRYPTION_KEY).
ALTER TABLE payment_gateway_config
  ADD COLUMN elproof_app_id VARCHAR(60) NULL AFTER midtrans_server_key_enc,
  ADD COLUMN elproof_secret_enc VARBINARY(500) NULL AFTER elproof_app_id,
  ADD COLUMN elproof_base_url VARCHAR(255) NOT NULL DEFAULT 'https://elproof.elcodelabs.com/api/v1' AFTER elproof_secret_enc;

ALTER TABLE subscription_invoices MODIFY COLUMN provider ENUM('tripay','midtrans','elproof') NOT NULL;
