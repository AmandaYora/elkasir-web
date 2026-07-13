-- Billing tenant (toko) ke platform elkasir — TERPISAH TOTAL dari self_orders/payments
-- (pembayaran pelanggan toko). Modul `subscription` adalah satu-satunya pemilik 3 tabel ini;
-- ia memakai gateway QRIS yang SAMA (module `payment`) lewat paymentclient.Client, seperti
-- selforder, tapi tidak berbagi baris/tabel dengan selforder sama sekali (no piggyback).

-- Katalog paket langganan (data referensi, bukan data tenant — diisi via seed, bukan tenant CRUD).
CREATE TABLE subscription_plans (
  id          CHAR(26)     NOT NULL,
  code        VARCHAR(40)  NOT NULL,
  name        VARCHAR(100) NOT NULL,
  price       BIGINT       NOT NULL,
  period_days INT          NOT NULL DEFAULT 30,
  is_active   TINYINT(1)   NOT NULL DEFAULT 1,
  -- renewal_only: paket ini hanya bisa diperpanjang (checkout dengan plan_id yang SAMA dengan
  -- langganan berjalan) — tidak bisa dipilih pertama kali maupun jadi tujuan pindah/upgrade dari
  -- paket lain. Sengaja TIDAK ada di CreateSubscriptionPlan/UpdateSubscriptionPlan (lihat
  -- db/queries/subscriptions.sql) sehingga tidak bisa diubah lewat form Konsol Platform — sama
  -- seperti `code`, ini properti struktural yang ditetapkan sekali saat paket dibuat (migrasi).
  renewal_only TINYINT(1)  NOT NULL DEFAULT 0,
  created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_subscription_plans_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Status langganan toko saat ini (1 baris per store). plan_id = primitive ID lokal (FK
-- intra-modul ke subscription_plans, sah — keduanya dimiliki modul yang sama).
CREATE TABLE store_subscriptions (
  id                    CHAR(26) NOT NULL,
  store_id              CHAR(26) NOT NULL,
  plan_id               CHAR(26) NOT NULL,
  status                ENUM('trial','active','past_due','expired','canceled') NOT NULL DEFAULT 'trial',
  current_period_start  DATETIME NULL,
  current_period_end    DATETIME NULL,
  created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_store_subscriptions_store (store_id),
  KEY idx_store_subscriptions_plan (plan_id),
  CONSTRAINT fk_store_subscriptions_store FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE CASCADE,
  CONSTRAINT fk_store_subscriptions_plan FOREIGN KEY (plan_id) REFERENCES subscription_plans (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Riwayat & rekonsiliasi tagihan langganan lewat gateway QRIS (ledger MILIK modul subscription
-- sendiri — analog `payments` milik selforder, tapi tabel terpisah agar data dua domain bisnis
-- ini tidak pernah tercampur).
CREATE TABLE subscription_invoices (
  id            CHAR(26)     NOT NULL,
  store_id      CHAR(26)     NOT NULL,
  plan_id       CHAR(26)     NOT NULL,
  amount        BIGINT       NOT NULL DEFAULT 0,
  status        ENUM('pending','paid','expired','failed') NOT NULL DEFAULT 'pending',
  provider      ENUM('tripay','midtrans') NOT NULL,
  provider_ref  VARCHAR(190) NULL,
  period_start  DATETIME     NULL,
  period_end    DATETIME     NULL,
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_subscription_invoices_store (store_id),
  KEY idx_subscription_invoices_provider_ref (provider_ref),
  CONSTRAINT fk_subscription_invoices_store FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE CASCADE,
  CONSTRAINT fk_subscription_invoices_plan FOREIGN KEY (plan_id) REFERENCES subscription_plans (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
