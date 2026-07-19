ALTER TABLE payment_charge_apps ADD COLUMN provider_ref VARCHAR(191) NULL AFTER app_id;
ALTER TABLE payment_clients ADD COLUMN secret_enc VARBINARY(500) NULL AFTER secret_hash;
