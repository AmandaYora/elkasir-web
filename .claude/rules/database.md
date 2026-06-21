# Rule: Database (applies to `apps/api/db/**` and any repository code)

- Engine: **MySQL 8** at host/OS level (not a default Docker container). IDs are **ULID** strings.
- Migrations via **golang-migrate** in `db/migrations` (`NNNNNN_name.up.sql` / `.down.sql`). Create with
  `npm run migrate:create -- <name>`; apply with `npm run migrate:up`.
- Queries via **sqlc** in `db/queries/*.sql`; generated to `internal/platform/db/sqlcgen`
  (`npm run sqlc:generate`). **No GORM, no AutoMigrate, no ORM schema generation.**
- A table is **owned by exactly one module**; only that module's repository reads/writes it.
- **No cross-module foreign keys. No cross-module joins.** Cross-module references are stored as
  **primitive IDs** (e.g. `transactions.product_id`) and resolved via the owning module's contract
  client — not SQL joins.
- Every tenant-scoped table has `store_id`; queries always filter by it.
- Keep money as integer minor units (no floats). Timestamps stored as UTC datetimes (`parseTime`,
  `loc=UTC`).
