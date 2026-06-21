# Architecture Overview

Elkasir is a **modular monolith** backend (`apps/api`, Go) plus a **React 19 SPA** (`apps/web`),
shipped as **one container** (SPA embedded into the Go binary). MySQL runs at host level.

## Repository layout

```txt
elkasir_web/
├── apps/
│   ├── api/                     # Go modular-monolith backend
│   │   ├── cmd/api/             # entrypoint (composition root via internal/app)
│   │   ├── cmd/seed/            # demo data seeder
│   │   ├── internal/
│   │   │   ├── app/             # composition root: wires modules + router
│   │   │   ├── modules/<m>/     # one folder per module (see below)
│   │   │   ├── platform/        # shared technical layer (config, db/sqlcgen, httpserver, httpx, id, uow)
│   │   │   └── webui/           # embeds the built SPA (serves it at root)
│   │   ├── db/migrations/       # golang-migrate SQL (sqlc reads schema here)
│   │   └── db/queries/          # sqlc queries
│   └── web/                     # React 19 + Tailwind 4 SPA (src/{app,modules,shared,theme,styles})
├── packages/
│   ├── api-contract/            # OpenAPI spec → generated TS client
│   └── shared/                  # domain-agnostic TS shared across apps
├── docs/                        # human docs (PRD, SYSTEM_DESIGN, API_CONTRACT, DB_SCHEMA, DEPLOYMENT)
├── knowledge/                   # this knowledge base (AI + dev)
├── .claude/rules/               # path-scoped editing rules
├── infra/{docker,nginx}/        # one-container Dockerfile + optional reverse proxy
├── scripts/                     # cross-platform helpers (embed-web, migrate)
└── docker-compose.yml           # one app container + host MySQL
```

## Backend module anatomy

Every module under `apps/api/internal/modules/<module>/`:

```txt
modules/product/
├── contracts/            # PUBLIC boundary: client interface + DTOs + sentinel errors
├── application/          # use cases / services, input validation
├── domain/               # entities, value objects, domain rules
│   └── events/           # domain events (concept must exist; not every module emits)
├── infrastructure/       # repositories (sqlc / database/sql), contract implementation
├── presentation/         # HTTP handlers + route registration
└── product.module.go     # wiring: assembles repo → service → handler
```

**Only `contracts/` is importable by other modules.** A module never imports another module's
`application`, `infrastructure`, `domain`, or `presentation`.

## Boundaries (hard rules)

- Cross-module relations are stored as **primitive IDs** (`orders.product_id`), never physical
  foreign keys across modules.
- **No cross-module joins.** To read another module's data, call its contract client
  (e.g. `productclient.Client.GetForSale`).
- `auth` is a core module; other modules must not query auth/user tables directly — they consume
  the auth middleware/principal contract for protection and identity.

## Cross-module flows (orchestration)

Atomic cross-module flows use a **Unit-of-Work** (`internal/platform/uow`) so multiple modules'
writes share one DB transaction. Two orchestrators do this:

- **Cashier transaction** (`transaction`): check stock + record sale + decrement stock + update shift
  totals via `productclient`, `shiftclient`, `salesclient`.
- **Self-order checkout** (`selforder`): orchestrates `productclient`, `salesclient`, `shiftclient`,
  `tableclient`, `paymentclient`.

Each contract client is tx-aware: when invoked inside a UoW, its queries run on the shared
transaction (`uow.Q(ctx)`).

## Request/data flow

```txt
Browser (SPA)  ──HTTP /api/v1/*──▶  go-chi router (internal/app)
                                      └─▶ module presentation (handler)
                                            └─▶ application (service)  ──▶ contract clients (other modules)
                                                  └─▶ infrastructure (repo / sqlc)  ──▶ MySQL
SPA static files  ◀──root /*──  internal/webui (embedded)
```

Responses use the standard envelope (`{success,message,data}` / paginated `meta` / `{success,message,errors}`) — see [API_GUIDE.md](API_GUIDE.md).

## Deployment topology

```txt
1 host (VPS)
├── MySQL 8 at OS level (serves many DBs)
└── Docker
    └── one app container
        ├── React SPA (static, embedded)  → served at /
        └── Go API                         → served at /api/v1
```

The container reaches the host DB via `host.docker.internal`. See [docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md).
