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
| `000004_selforder_payments` | `self_orders`, `self_order_items`, `payments` | Kondisi 2 & 3; tautan silang `transactions.self_order_id` ↔ `self_orders.transaction_id`. `payments` dimiliki modul `selforder` (bukan `payment` — lihat DATABASE_GUIDE.md §4) |
| `000015_subscription_billing` | `subscription_plans`, `store_subscriptions`, `subscription_invoices` | Billing tenant→platform (modul `subscription`), domain TERPISAH dari self_orders/payments meski memakai gateway QRIS yang sama |
| `000016_platform` | `platform_users`; kolom baru `stores.slug`/`stores.status` | Superadmin + siklus hidup tenant (modul `platform`). Juga 2 fix multi-tenancy: `staff.username` jadi unik GLOBAL (login POS lintas-tenant tanpa konteks toko), dan self-order QR wajib sertakan slug toko (`dining_tables.code` cuma unik per-toko) |
| `000017_withdrawal_processing` | kolom baru `withdrawals.processed_by`/`claimed_at`/`processed_at`/`rejected_reason` | Alur klaim→selesai (pending→processing→success/failed) oleh superadmin (modul `withdrawal`) |
| `000018_subscription_legacy_backfill` | data seed: plan `legacy-grandfather` (`is_active=0`) + backfill `store_subscriptions` | Migrasi data satu kali — tenant lama diberi paket "warisan" 20 tahun supaya gerbang langganan (§2.15) tidak mengunci mereka retroaktif |
| `000019_payment_gateway_registry` | `payment_clients`, `payment_gateway_config`, `payment_charge_apps` | Registry app + config gateway ter-DB-kan (enkripsi AES-256-GCM) + indeks dispatch webhook — menggantikan trik prefix `"sub_"` (PLAN.md §9, Part 2). Tetap SATU dompet gateway; `payment_clients` di-seed dengan `ELKASIR-SELFORDER`/`ELKASIR-SUBSCRIBE` |
| `000020_payment_external_api` | kolom baru `payment_clients.secret_enc` | Salinan reversibel (AES-256-GCM) dari plaintext yang sama dengan `secret_hash` (bcrypt) — dipakai HANYA untuk menandatangani relay webhook keluar ke app `kind='external'` (PLAN.md §10.1.6, Part 3) |
| `000021_payment_charge_provider_ref` | kolom baru `payment_charge_apps.provider_ref` | Menerjemahkan `orderRef` milik pemanggil eksternal menjadi `provider_ref` gateway, dipakai endpoint status eksternal (PLAN.md §10.2 EB2, Part 3) |

## Relasi kunci (3 kondisi order)

- **Kondisi 1** (kasir): `transactions(source='cashier', table_id=NULL, self_order_id=NULL)`.
- **Kondisi 2** (QRIS): `self_orders(payment_method='qris', payment_status:pending→paid)`
  → saat `paid`: buat `transactions(source='self_order', payment_method='qris')`, kurangi stok.
- **Kondisi 3** (bayar di kasir): `self_orders(payment_method='cash', payment_status='unpaid', claim_code UNIQUE)`
  → saat ditebus: buat `transactions(source='self_order', payment_method='cash')`, kurangi stok, `paid`+`completed`.

> Stok dikurangi **saat pembayaran terkonfirmasi** (webhook QRIS / redeem tunai), bukan saat order dibuat.
