ALTER TABLE subscription_invoices MODIFY COLUMN provider ENUM('tripay','midtrans') NOT NULL;
ALTER TABLE payment_gateway_config
  DROP COLUMN elproof_base_url,
  DROP COLUMN elproof_secret_enc,
  DROP COLUMN elproof_app_id;
