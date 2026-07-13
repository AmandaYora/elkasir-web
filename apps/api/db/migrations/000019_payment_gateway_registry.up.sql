-- Part 2 (PLAN.md §9, PB0): payment module gains a formal "app" registry (replacing the ad-hoc
-- "sub_" order-ref prefix convention) and DB-backed gateway config (replacing `.env`-at-boot).
-- Still exactly ONE active gateway/merchant account ("one wallet", §9.1.1) — these tables never
-- hold per-app credentials, only per-app identity + attribution.

-- Registry of apps allowed to create charges through the one shared gateway. `secret_hash` is
-- NULL for `kind='internal'` rows — internal consumers call CreateCharge as direct in-process Go
-- calls (no network hop to authenticate across, §9.1.9); it's populated only when a `kind
-- ='external'` row is created (§9.1.3/§9.1.11), and enforced only once the external API (§9.7)
-- exists. `callback_url` is likewise only meaningful for `kind='external'`.
CREATE TABLE payment_clients (
  id            CHAR(26)     NOT NULL,
  app_id        VARCHAR(60)  NOT NULL,
  name          VARCHAR(150) NOT NULL,
  secret_hash   VARCHAR(100) NULL,
  kind          ENUM('internal','external') NOT NULL,
  callback_url  VARCHAR(500) NULL,
  status        ENUM('active','inactive') NOT NULL DEFAULT 'active',
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_payment_clients_app_id (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Single-row config table (enforced by application convention, not a DB constraint — same
-- shape as `settings`' effectively-one-row-per-store pattern, just one row total here since
-- there's exactly one wallet). Secret fields are stored ENCRYPTED (AES-256-GCM, application-level,
-- key from CONFIG_ENCRYPTION_KEY env — §9.1.2) — never plaintext, never returned to the browser.
CREATE TABLE payment_gateway_config (
  id                        CHAR(26)     NOT NULL,
  provider                  VARCHAR(20)  NOT NULL DEFAULT '',
  sandbox                   TINYINT(1)   NOT NULL DEFAULT 1,
  tripay_api_key_enc        VARBINARY(500) NULL,
  tripay_private_key_enc    VARBINARY(500) NULL,
  tripay_merchant_code_enc  VARBINARY(500) NULL,
  tripay_method             VARCHAR(30)  NOT NULL DEFAULT 'QRIS',
  midtrans_server_key_enc   VARBINARY(500) NULL,
  created_at                DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at                DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Thin index recording which app created a given gateway charge (§9.1.4) — NOT a business
-- ledger (payment still owns no ledger of its own, unchanged principle from Part 1). Used only
-- so an incoming webhook's order_ref can be looked up to find which app/consumer owns it,
-- replacing the "sub_" prefix string-sniffing hack.
CREATE TABLE payment_charge_apps (
  order_ref   VARCHAR(191) NOT NULL,
  app_id      VARCHAR(60)  NOT NULL,
  created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (order_ref)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Seed the two known internal consumers (§9.1.3) — same migration-seeds-a-reference-row
-- precedent as `legacy-grandfather` (migration 000018). IDs use RANDOM_BYTES(13) hex-encoded
-- (26 chars) — migrations have no access to the app's Go ULID generator.
INSERT INTO payment_clients (id, app_id, name, secret_hash, kind, callback_url, status)
SELECT LOWER(HEX(RANDOM_BYTES(13))), 'ELKASIR-SELFORDER', 'Elkasir Self-Order', NULL, 'internal', NULL, 'active'
WHERE NOT EXISTS (SELECT 1 FROM payment_clients WHERE app_id = 'ELKASIR-SELFORDER');

INSERT INTO payment_clients (id, app_id, name, secret_hash, kind, callback_url, status)
SELECT LOWER(HEX(RANDOM_BYTES(13))), 'ELKASIR-SUBSCRIBE', 'Elkasir Subscribe', NULL, 'internal', NULL, 'active'
WHERE NOT EXISTS (SELECT 1 FROM payment_clients WHERE app_id = 'ELKASIR-SUBSCRIBE');
