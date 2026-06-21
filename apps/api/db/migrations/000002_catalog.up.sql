-- Fase 2 — katalog: kategori, produk, meja.

CREATE TABLE product_categories (
  id          CHAR(26)     NOT NULL,
  store_id    CHAR(26)     NOT NULL,
  name        VARCHAR(120) NOT NULL,
  sort_order  INT          NOT NULL DEFAULT 0,
  created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_categories_store_name (store_id, name),
  CONSTRAINT fk_categories_store FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE products (
  id           CHAR(26)     NOT NULL,
  store_id     CHAR(26)     NOT NULL,
  category_id  CHAR(26)     NULL,
  sku          VARCHAR(60)  NULL,
  name         VARCHAR(150) NOT NULL,
  price        BIGINT       NOT NULL DEFAULT 0,
  cost         BIGINT       NOT NULL DEFAULT 0,
  stock        INT          NOT NULL DEFAULT 0,
  status       ENUM('active','inactive') NOT NULL DEFAULT 'active',
  image_url    VARCHAR(500) NULL,
  created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_products_store_sku (store_id, sku),
  KEY idx_products_store_status (store_id, status),
  KEY idx_products_category (category_id),
  CONSTRAINT fk_products_store FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE CASCADE,
  CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES product_categories (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Meja untuk self-order; `code` jadi isi QR (/order/<code>).
CREATE TABLE dining_tables (
  id          CHAR(26)     NOT NULL,
  store_id    CHAR(26)     NOT NULL,
  code        VARCHAR(40)  NOT NULL,
  name        VARCHAR(60)  NOT NULL,
  area        VARCHAR(60)  NOT NULL DEFAULT '',
  seats       INT          NOT NULL DEFAULT 0,
  status      ENUM('active','inactive') NOT NULL DEFAULT 'active',
  created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_tables_store_code (store_id, code),
  CONSTRAINT fk_tables_store FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
