# Database — Elkasir

MySQL 8. Migrasi via **golang-migrate** (`db/migrations`, pasangan `*.up.sql`/`*.down.sql`,
append-only). Query type-safe via **sqlc** (`db/queries` → `internal/platform/db/sqlcgen`).

```bash
task migrate          # terapkan semua migrasi
task migrate:down     # rollback 1 langkah
task seed             # data demo (idempoten)
task gen:sqlc         # regenerate kode Go dari skema + query
```

## Konvensi

- **ID**: ULID `CHAR(26)` (server-generated).
- **Uang**: `BIGINT` rupiah penuh (tanpa desimal).
- **Waktu**: `DATETIME` UTC.
- **Tenant**: setiap entitas bisnis ber-`store_id` (single-store dulu).
- **Snapshot**: `transaction_items` & `self_order_items` menyimpan `product_name`,
  `category`, `price` saat penjualan — kebal edit produk berikutnya.

## Tabel

| Migrasi | Tabel | Catatan |
| --- | --- | --- |
| `000001_core_identity` | `stores`, `settings`, `admin_users`, `staff`, `refresh_tokens`, `idempotency_keys`, `webhook_events` | identitas 2-konteks (admin & staff), policy kontrol, infra auth/idempotensi |
| `000002_catalog` | `product_categories`, `products`, `dining_tables` | master data; `dining_tables.code` = isi QR |
| `000003_operations` | `shifts`, `transactions`, `transaction_items`, `cash_movements`, `withdrawals` | inti POS; `transactions` ber-`source` (`cashier`/`self_order`), `shift_id`/`table_id`/`self_order_id` nullable |
| `000004_selforder_payments` | `self_orders`, `self_order_items`, `payments` | Kondisi 2 & 3; tautan silang `transactions.self_order_id` ↔ `self_orders.transaction_id` |

## Relasi kunci (3 kondisi order)

- **Kondisi 1** (kasir): `transactions(source='cashier', table_id=NULL, self_order_id=NULL)`.
- **Kondisi 2** (QRIS): `self_orders(payment_method='qris', payment_status:pending→paid)`
  → saat `paid`: buat `transactions(source='self_order', payment_method='qris')`, kurangi stok.
- **Kondisi 3** (bayar di kasir): `self_orders(payment_method='cash', payment_status='unpaid', claim_code UNIQUE)`
  → saat ditebus: buat `transactions(source='self_order', payment_method='cash')`, kurangi stok, `paid`+`completed`.

> Stok dikurangi **saat pembayaran terkonfirmasi** (webhook QRIS / redeem tunai), bukan saat order dibuat.
