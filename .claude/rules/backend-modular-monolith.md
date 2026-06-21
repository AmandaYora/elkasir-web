# Rule: Backend Modular Monolith (applies to `apps/api/**`)

- Each module lives in `internal/modules/<module>/` with: `contracts/`, `application/`, `domain/`
  (+`domain/events/`), `infrastructure/`, `presentation/`, and a `<module>.module.go` wiring file.
- **Only `contracts/` is public.** Never import another module's `application`, `infrastructure`,
  `domain`, or `presentation`. Cross-module access goes through the provider's `contracts` client.
- Contracts are owned by the **provider** module (the capability), not the consumer.
- A repository may only touch tables owned by its own module. **No cross-module joins. No cross-module
  foreign keys.** Cross-module relations are **primitive IDs**; validate/lookup via contract clients.
- Atomic cross-module flows use the Unit-of-Work (`internal/platform/uow`); contract clients run on the
  shared transaction via `uow.Q(ctx)`. Never call another module's repository directly.
- `auth` is a core module: get identity from the auth principal/middleware; do not query auth/user
  tables from other modules.
- DB access: **sqlc + database/sql + golang-migrate**. **No GORM, no AutoMigrate.** Generated code →
  `internal/platform/db/sqlcgen` (`npm run sqlc:generate`).
- The dev watcher is **Air** (`npm run dev:api`). Don't make `go run ./cmd/api` the default loop.
- Multi-tenancy: derive `store_id` from the authenticated principal, never from the request body; every
  query filters by `store_id`.
- HTTP responses/errors go through `internal/platform/httpx` helpers so the standard envelope is
  applied centrally. Endpoints are mounted under `/api/v1`.
