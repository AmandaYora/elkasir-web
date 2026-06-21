# Elkasir — System Design

> Architecture of the Elkasir monorepo: repository layout, the Go modular-monolith
> backend, the React web frontend, module boundary rules, request/data flow, the
> one-container deployment topology, and the standard API response envelopes.
> For product context see [`PRD.md`](./PRD.md); for ops see [`DEPLOYMENT.md`](./DEPLOYMENT.md).

---

## 1. High-level overview

Elkasir is a **modular monolith**. One Go binary serves everything:

- the **React 19 SPA** (admin dashboard + customer self-order pages) at the root
  path, served from assets **embedded into the binary**, and
- the **REST API** under `/api/v1`.

There is **no separate web server** in production and **no microservices** — the
"modules" are logical boundaries inside one process. State lives in a **host-level
MySQL 8** database. A separate Flutter cashier app (`elkasir_pos`) consumes the
same API but is not part of this repo.

Stack (locked):

- **Backend** (`apps/api`): Go, [go-chi] router, MySQL via **sqlc +
  golang-migrate + database/sql** (no ORM), **ULID** IDs, **JWT** auth.
- **Frontend** (`apps/web`): **React 19 + Tailwind 4**, **react-router-dom** (lazy
  routes), **Zustand** (state), **Zod** (validation), **Axios** (HTTP). No
  TanStack.
- **Tooling**: plain **npm workspaces** (no Turborepo); OpenAPI contract →
  TypeScript types.

---

## 2. Monorepo layout

```
elkasir_web/
├── apps/
│   ├── web/                 # React 19 SPA: admin dashboard + self-order pages (Vite, dev :8080)
│   └── api/                 # Go modular-monolith backend (:8081); serves /api/v1 + embedded SPA
├── packages/
│   ├── api-contract/        # OpenAPI spec → generated TypeScript client/types (@elkasir/api-contract)
│   └── shared/              # domain-agnostic shared TS utilities (@elkasir/shared)
├── docs/                    # human-facing docs (this file, PRD.md, DEPLOYMENT.md)
├── infra/
│   ├── docker/Dockerfile    # one-container build (SPA → embed → Go binary)
│   └── nginx/elkasir.conf   # optional host reverse proxy (TLS termination)
├── scripts/
│   ├── embed-web.mjs        # copies built SPA into the Go embed dir
│   └── migrate.mjs          # golang-migrate wrapper (migrate:up/down/create)
├── docker-compose.yml       # one app container + host MySQL via host.docker.internal
├── package.json             # npm workspaces + the canonical scripts
└── .env.example             # documented environment variables
```

Only `apps/web` and `packages/*` are npm workspaces. **`apps/api` is a Go module**,
not an npm workspace.

---

## 3. Backend: modular-monolith design (`apps/api`)

### 3.1 Module structure

Every business capability is a **module** under `apps/api/internal/modules/<module>/`
with a fixed internal shape:

```
apps/api/internal/modules/<module>/
├── contracts/          # PUBLIC surface: interfaces + DTOs other modules may import
├── application/        # use-cases / services (orchestration of domain + repos)
├── domain/             # entities, value objects, business rules (+ events/ where used)
├── infrastructure/     # repositories — sqlc-backed persistence, external clients
├── presentation/       # HTTP handlers (go-chi), request/response mapping
└── <module>.module.go  # wiring: build repo → service → handler, register routes
```

Modules (mirroring the product feature set): `auth`, `product`, `category`,
`table`, `staff`, `adminuser`, `transaction`, `shift`, `cashmovement`,
`withdrawal`, `report`, `selforder`, `payment`.

### 3.2 Shared platform

Cross-cutting, domain-agnostic code lives in `apps/api/internal/platform/`:

- `config/` — environment/config loading (builds the MySQL DSN from `DB_*`).
- `db/` — connection pool (`database/sql`) and the generated `sqlcgen/` queries.
- `httpx/` — standard response/error envelopes, decoding, pagination helpers.
- `httpserver/` — chi router construction, middleware, health endpoints.
- `uow/` — the **Unit-of-Work** manager for cross-module atomic flows.
- `id/` — ULID generation.

The **composition root** wires config → DB pool → modules → router, then mounts
health at the root, the API under `/api/v1`, and the embedded SPA as a catch-all.

### 3.3 Module boundary rules

These are the non-negotiable rules that keep the monolith modular:

1. **Contracts-only imports.** A module may import *only* the `contracts/` package
   of another module. Importing another module's `application/`, `domain/`,
   `infrastructure/`, or `presentation/` is forbidden.
2. **Primitive-ID relations.** Cross-module references are stored as plain IDs
   (e.g. a transaction holds `tableId`, `shiftId`, `cashierId`, `selfOrderId` as
   strings). No module embeds another module's entity type.
3. **No cross-module DB joins or foreign keys.** Each module owns its tables.
   A module never `JOIN`s onto another module's tables; it resolves the related
   data by calling the owning module's contract client.
4. **Multi-tenancy is enforced at the boundary.** Every row is scoped by
   `store_id`, which is always taken from the authenticated principal — never from
   the request body.
5. **Unit-of-Work for cross-module atomic flows.** When one operation must mutate
   several modules together (e.g. **self-order checkout** = self-order +
   transaction + shift totals, or a **cashier sale** = transaction + stock
   decrement + shift totals), the orchestrating module runs the contract clients
   inside a single `uow` transaction so the whole flow commits or rolls back
   atomically.

```
   transaction module (orchestrator)
        │  uses contract clients (tx-aware via UoW)
        ├──► product.Client    (decrement stock)
        ├──► shift.Client       (roll sale into shift totals)
        └──► transaction.SalesClient (persist sale)
              ── all inside one uow.Tx → commit or rollback together ──
```

---

## 4. Frontend: web app structure (`apps/web`)

```
apps/web/src/
├── app/         # app shell, router setup, providers, lazy route definitions
├── modules/     # one folder per feature: pages, components, services, schemas, stores, hooks, types
├── shared/      # generic UI + cross-feature code; HTTP goes through shared/services/http-client.ts (Axios)
├── theme/       # centralized theme colors / design tokens
└── styles/      # global styles (Tailwind 4)
```

Key conventions:

- **Lazy routes** via `react-router-dom` — each feature's pages are code-split.
- **State** with **Zustand** stores (e.g. auth, cart) — no TanStack Query.
- **Validation** with **Zod** schemas at the module boundary.
- **All HTTP** goes through a single Axios **`http-client`** in
  `src/shared/services/` (base URL from `VITE_API_BASE_URL`, attaches the JWT,
  handles refresh/error mapping centrally). Feature services call the http-client;
  components never call Axios directly.
- **Theme** colors are centralized in `src/theme/` rather than scattered across
  components.

The same SPA hosts both the **admin dashboard** (sidebar groups: Overview —
dashboard/products/categories/transactions; Operations — incoming
orders/shifts/tables/cash movements/withdrawals; Analytics —
statistics/staff/users) and the **customer self-order pages** (table-code routes,
unauthenticated).

---

## 5. Request / data flow

A typical authenticated request from the admin dashboard:

```
Browser (React SPA, Axios http-client)
   │  HTTPS, Authorization: Bearer <JWT>
   ▼
/api/v1/<resource>                          ← API base path
   │
   ▼
chi router (httpserver)
   │  middleware: CORS, request-id, recover, JWT auth → principal (store_id, role)
   ▼
<module>.presentation  (HTTP handler)
   │  decode + validate request; never trusts store_id from body
   ▼
<module>.application  (service / use-case)
   │  business rules; for cross-module atomic work, opens a UoW transaction
   │  and calls other modules' contract clients
   ▼
<module>.infrastructure  (repository, sqlc-generated queries)
   │
   ▼
MySQL 8  (host-level)
```

The response travels back up and is written using the **standard envelope**
(section 7). The SPA itself is served by the **same binary**: any non-`/api/v1`,
non-health path falls through to the embedded-SPA catch-all handler, which serves
static assets or `index.html` for client-side routes.

Health/liveness lives at the **root**, outside `/api/v1`:

- `GET /healthz` — liveness (process is up).
- `GET /readyz` — readiness (DB reachable).

---

## 6. Deployment topology (one container)

Production runs as **one application container** plus a **host-level MySQL**
(MySQL is *not* containerized by default — one MySQL instance can serve many
databases/apps).

```
                         ┌─────────────────────────────────────────┐
                         │                Host / VPS                │
                         │                                          │
  Internet ──► :80/:443 ─┤  nginx / Caddy (optional, TLS) ──┐       │
                         │                                  │       │
                         │   ┌──────────────────────────────▼────┐  │
                         │   │  Docker container: elkasir-app     │  │
                         │   │  (single Go binary)                │  │
                         │   │   • serves SPA at  /     (embedded)│  │
                         │   │   • serves API at  /api/v1         │  │
                         │   │   listens 127.0.0.1:8081           │  │
                         │   └──────────────┬─────────────────────┘  │
                         │                  │ host.docker.internal    │
                         │                  ▼                         │
                         │        ┌───────────────────┐               │
                         │        │  MySQL 8 (host OS) │  :3306        │
                         │        └───────────────────┘               │
                         └─────────────────────────────────────────┘
```

How the single binary is produced (see [`DEPLOYMENT.md`](./DEPLOYMENT.md)):

1. `apps/web` is built to static assets (Vite).
2. `scripts/embed-web.mjs` copies those assets into the Go embed directory
   (`internal/webui/dist`); `go:embed` bakes them into the binary.
3. `apps/api` is compiled — the resulting binary serves both SPA + API.

The container reaches the host MySQL via `host.docker.internal` (mapped to
`host-gateway` in `docker-compose.yml`). The container listens on
`127.0.0.1:8081`; a host reverse proxy terminates TLS and forwards to it.

---

## 7. Standard response envelopes

All API responses use a small, stable set of shapes (defined in
`internal/platform/httpx`).

### 7.1 Single resource
A `200/201` returns the resource object directly (camelCase fields), e.g.:

```json
{
  "id": "01HZX…",
  "name": "Es Kopi Susu",
  "price": 18000,
  "status": "active"
}
```

### 7.2 List (paginated)
List endpoints wrap items in a pagination envelope; `data` is never `null`:

```json
{
  "data": [ { "id": "01HZX…", "name": "Es Kopi Susu" } ],
  "pagination": { "total": 128, "limit": 20, "offset": 0 }
}
```

Pagination is driven by `?limit` & `?offset` (or `?page`), bounded by a safe
maximum limit.

### 7.3 Error
Errors use a single envelope with a **stable machine-readable code**:

```json
{
  "error": {
    "code": "validation_error",
    "message": "price must be greater than 0",
    "details": { "field": "price" }
  }
}
```

Stable error codes and their HTTP statuses:

| Code | HTTP | Meaning |
|---|---|---|
| `bad_request` | 400 | Malformed request |
| `validation_error` | 400 | Request failed validation (details carries field errors) |
| `unauthorized` | 401 | Missing/invalid credentials |
| `forbidden` | 403 | Authenticated but not allowed |
| `not_found` | 404 | Resource does not exist (within the tenant) |
| `conflict` | 409 | State conflict (e.g. duplicate) |
| `unprocessable` | 422 | Semantically invalid |
| `rate_limited` | 429 | Too many requests |
| `internal` | 500 | Unexpected server error |

`details` is optional and omitted when empty.

---

## 8. Auth model (summary)

JWT-based auth with **two principal types**:

- **admin** — web dashboard users (`owner`/`admin`/`manager`/`viewer`), email login.
- **staff** — POS users (`cashier`/`supervisor`), username login, used by the
  Flutter app.

Login returns `accessToken`, `refreshToken`, `expiresIn`, and the principal
(`AuthUser` with `storeId`, `role`, `actor`). The access-token TTL and
refresh-token TTL are configurable (`JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`). The
`storeId` carried in the token is the single source of truth for tenant scoping.

[go-chi]: https://github.com/go-chi/chi
