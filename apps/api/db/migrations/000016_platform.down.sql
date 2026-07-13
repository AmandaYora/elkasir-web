ALTER TABLE refresh_tokens MODIFY COLUMN actor ENUM('admin','staff') NOT NULL;

ALTER TABLE staff DROP INDEX uq_staff_username;
ALTER TABLE staff ADD UNIQUE KEY uq_staff_store_username (store_id, username);

ALTER TABLE stores DROP INDEX uq_stores_slug;
ALTER TABLE stores DROP COLUMN status;
ALTER TABLE stores DROP COLUMN slug;

DROP TABLE IF EXISTS platform_users;
