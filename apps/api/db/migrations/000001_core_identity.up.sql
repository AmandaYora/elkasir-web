-- Fase 2 — inti & identitas.
-- Konvensi: ID = ULID CHAR(26) (dihasilkan server); uang = BIGINT rupiah penuh;
-- waktu = DATETIME UTC; semua entitas bisnis ber-store_id (multi-tenant, single-store dulu).

CREATE TABLE stores (
  id          CHAR(26)     NOT NULL,
  name        VARCHAR(150) NOT NULL,
  type        VARCHAR(60)  NOT NULL DEFAULT 'F&B',
  address     VARCHAR(255) NULL,
  phone       VARCHAR(40)  NULL,
  timezone    VARCHAR(64)  NOT NULL DEFAULT 'Asia/Jakarta',
  currency    CHAR(3)      NOT NULL DEFAULT 'IDR',
  created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Kebijakan kontrol + profil + feature flags (1 baris per store).
CREATE TABLE settings (
  id                       CHAR(26) NOT NULL,
  store_id                 CHAR(26) NOT NULL,
  max_discount_percent     INT      NOT NULL DEFAULT 10,
  max_operational_expense  BIGINT   NOT NULL DEFAULT 200000,
  cash_variance_tolerance  BIGINT   NOT NULL DEFAULT 5000,
  feature_self_order       TINYINT(1) NOT NULL DEFAULT 1,
  feature_qris             TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_settings_store (store_id),
  CONSTRAINT fk_settings_store FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Pengguna admin (dashboard web). Beda konteks dari staff (POS).
CREATE TABLE admin_users (
  id             CHAR(26)     NOT NULL,
  store_id       CHAR(26)     NOT NULL,
  name           VARCHAR(150) NOT NULL,
  email          VARCHAR(190) NOT NULL,
  password_hash  VARCHAR(100) NOT NULL,
  role           ENUM('owner','admin','manager','viewer') NOT NULL DEFAULT 'viewer',
  status         ENUM('active','inactive') NOT NULL DEFAULT 'active',
  last_active_at DATETIME     NULL,
  created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_admin_users_email (email),
  KEY idx_admin_users_store (store_id),
  CONSTRAINT fk_admin_users_store FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Staf POS (kasir/supervisor). Login via username di aplikasi Flutter.
CREATE TABLE staff (
  id             CHAR(26)     NOT NULL,
  store_id       CHAR(26)     NOT NULL,
  name           VARCHAR(150) NOT NULL,
  username       VARCHAR(100) NOT NULL,
  email          VARCHAR(190) NULL,
  password_hash  VARCHAR(100) NOT NULL,
  role           ENUM('cashier','supervisor') NOT NULL DEFAULT 'cashier',
  status         ENUM('active','inactive') NOT NULL DEFAULT 'active',
  created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_staff_store_username (store_id, username),
  KEY idx_staff_store (store_id),
  CONSTRAINT fk_staff_store FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Refresh token (rotasi) untuk kedua konteks auth.
CREATE TABLE refresh_tokens (
  id          CHAR(26) NOT NULL,
  actor       ENUM('admin','staff') NOT NULL,
  subject_id  CHAR(26) NOT NULL,
  token_hash  CHAR(64) NOT NULL,
  expires_at  DATETIME NOT NULL,
  revoked_at  DATETIME NULL,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_refresh_tokens_hash (token_hash),
  KEY idx_refresh_tokens_subject (actor, subject_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Idempotency untuk pembuatan transaksi/redeem dari klien.
CREATE TABLE idempotency_keys (
  id               CHAR(26)     NOT NULL,
  store_id         CHAR(26)     NOT NULL,
  idempotency_key  VARCHAR(255) NOT NULL,
  request_hash     CHAR(64)     NOT NULL,
  response_status  INT          NULL,
  response_body    LONGTEXT     NULL,
  created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_idempotency_store_key (store_id, idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Idempotensi webhook (event provider bisa terkirim berulang).
CREATE TABLE webhook_events (
  id           CHAR(26)     NOT NULL,
  provider     VARCHAR(40)  NOT NULL,
  event_id     VARCHAR(255) NOT NULL,
  processed_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_webhook_provider_event (provider, event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
