# Backend Guide (`apps/api`)

Go modular monolith. Module boundary rules are in [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)
and [MODULE_MAP.md](MODULE_MAP.md); this file is the day-to-day convention reference.

## Stack

- Router: **go-chi v5**. IDs: **ULID** (`internal/platform/id`). Auth: **JWT** (golang-jwt v5).
- DB: **MySQL** via **`database/sql`** + go-sql-driver, **sqlc** for type-safe queries, **golang-migrate** for schema. **No GORM, no AutoMigrate.**
- Config: 12-factor from env via `internal/platform/config` (`godotenv` loads `.env` in dev).

## Module layout

```txt
internal/modules/<module>/
├── contracts/        package <module>client  — interface + DTOs + sentinel errors (the ONLY public package)
├── application/      package application      — Service: use cases, validation, DTO mapping
├── domain/           package domain           — entities, value objects, input types, rules
│   └── events/       package events           — domain events (concept must exist)
├── infrastructure/   package infrastructure   — Repo (sqlc/database/sql) + contract implementation
├── presentation/     package presentation     — Handler + Routes(r chi.Router)
└── <module>.module.go  package <module>       — Module struct + New(...) wiring repo→service→handler
```

The composition root `internal/app/app.go` builds each module via its `<module>.module.go`
and mounts `Routes` under `/api/v1`.

## Conventions

- **Multi-tenancy**: read `storeID := auth.Principal(ctx).StoreID` in the handler; pass it down. Never
  trust a `storeId` from the request body. Every query filters by `store_id`.
- **Cross-module access**: only via a provider's `contracts` client. Never import another module's
  `infrastructure`/`application`/`domain`/`presentation`. Never JOIN across module tables.
- **Atomic cross-module writes**: open a Unit-of-Work (`internal/platform/uow`); contract clients
  invoked inside it run on the shared transaction (`uow.Q(ctx)`).
- **Errors**: return typed errors from `internal/platform/httpx` (`httpx.NotFound`, `Validation`,
  `Conflict`, …). Handlers call `httpx.Error(w, err)`; the envelope is produced centrally.
- **Responses**: handlers use `httpx` helpers (`httpx.OK`, `httpx.Created`, `httpx.List`,
  `httpx.NoContent`) → standard envelope. See [API_GUIDE.md](API_GUIDE.md).
- **sqlc**: write queries in `db/queries/*.sql`, run `npm run sqlc:generate`; generated code lands in
  `internal/platform/db/sqlcgen`. Dynamic/optional-filter queries may be hand-written in the repo with
  `database/sql`, still scoped to the module's own tables.

## Dev workflow

```bash
npm run dev:api            # Air live-reload (go install github.com/air-verse/air@latest once)
npm run migrate:create -- create_widgets_table
npm run migrate:up
npm run sqlc:generate
cd apps/api && go vet ./... && go test ./...
```

The dev watcher is **Air** — never `go run ./cmd/api` as the default loop.

## Adding a module

1. `internal/modules/<m>/` with the six parts above; define `contracts` first if other modules need it.
2. Add tables via a migration (own tables only; relations as primitive IDs).
3. Write `db/queries/<m>.sql`, `npm run sqlc:generate`.
4. Implement repo → service → handler; expose `Routes`.
5. Wire it in `internal/app/app.go` under the `/api/v1` group.
