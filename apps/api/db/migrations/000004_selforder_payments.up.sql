-- Fase 2 — self-order & pembayaran (Kondisi 2 & 3) + tautan silang ke transactions.

CREATE TABLE self_orders (
  id              CHAR(26)     NOT NULL,
  store_id        CHAR(26)     NOT NULL,
  table_id        CHAR(26)     NULL,
  status          ENUM('placed','preparing','completed') NOT NULL DEFAULT 'placed',
  payment_method  ENUM('qris','cash') NOT NULL,
  payment_status  ENUM('pending','paid','expired','failed','unpaid') NOT NULL,
  claim_code      VARCHAR(40)  NULL,
  subtotal        BIGINT       NOT NULL DEFAULT 0,
  total           BIGINT       NOT NULL DEFAULT 0,
  customer_note   VARCHAR(255) NULL,
  transaction_id  CHAR(26)     NULL,
  expires_at      DATETIME     NULL,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_self_orders_claim_code (claim_code),
  KEY idx_self_orders_store_status (store_id, status),
  KEY idx_self_orders_payment_status (payment_status),
  CONSTRAINT fk_self_orders_store FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE CASCADE,
  CONSTRAINT fk_self_orders_table FOREIGN KEY (table_id) REFERENCES dining_tables (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE self_order_items (
  id             CHAR(26)     NOT NULL,
  self_order_id  CHAR(26)     NOT NULL,
  product_id     CHAR(26)     NULL,
  product_name   VARCHAR(150) NOT NULL,
  category       VARCHAR(120) NOT NULL DEFAULT '',
  price          BIGINT       NOT NULL DEFAULT 0,
  quantity       INT          NOT NULL DEFAULT 0,
  line_total     BIGINT       NOT NULL DEFAULT 0,
  note           VARCHAR(255) NULL,
  created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_self_order_items_order (self_order_id),
  CONSTRAINT fk_self_order_items_order FOREIGN KEY (self_order_id) REFERENCES self_orders (id) ON DELETE CASCADE,
  CONSTRAINT fk_self_order_items_product FOREIGN KEY (product_id) REFERENCES products (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Riwayat & rekonsiliasi pembayaran gateway (Xendit).
CREATE TABLE payments (
  id             CHAR(26)     NOT NULL,
  store_id       CHAR(26)     NOT NULL,
  self_order_id  CHAR(26)     NOT NULL,
  provider       ENUM('xendit') NOT NULL DEFAULT 'xendit',
  provider_ref   VARCHAR(190) NULL,
  method         ENUM('qris') NOT NULL DEFAULT 'qris',
  amount         BIGINT       NOT NULL DEFAULT 0,
  status         ENUM('pending','paid','expired','failed') NOT NULL DEFAULT 'pending',
  raw_payload    LONGTEXT     NULL,
  created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_payments_self_order (self_order_id),
  KEY idx_payments_provider_ref (provider_ref),
  CONSTRAINT fk_payments_store FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE CASCADE,
  CONSTRAINT fk_payments_self_order FOREIGN KEY (self_order_id) REFERENCES self_orders (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Tautan silang (circular) ditambahkan setelah kedua tabel ada.
ALTER TABLE transactions
  ADD CONSTRAINT fk_transactions_self_order FOREIGN KEY (self_order_id) REFERENCES self_orders (id);

ALTER TABLE self_orders
  ADD CONSTRAINT fk_self_orders_transaction FOREIGN KEY (transaction_id) REFERENCES transactions (id);
