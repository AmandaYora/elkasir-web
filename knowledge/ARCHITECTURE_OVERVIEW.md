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
writes share one DB transaction. Orchestrators:

- **Cashier transaction** (`transaction`): check stock + record sale + decrement stock + update shift
  totals via `productclient`, `shiftclient`, `salesclient`.
- **Self-order checkout** (`selforder`): orchestrates `productclient`, `salesclient`, `shiftclient`,
  `tableclient`, `paymentclient`.
- **Subscription checkout** (`subscription`): creates a QRIS charge via `paymentclient` tagged
  with the `ELKASIR-SUBSCRIBE` app id — a separate business ledger from selforder's, even though
  both go through the same gateway.
- **Registry-driven webhook dispatch** (`payment`, Part 2): an incoming gateway callback resolves
  its `order_ref` to an `app_id` via a thin dispatch index, then either calls a registered
  in-process Go consumer directly (`kind=internal` — `selforder`/`subscription`) or fire-and-forgets
  a signed HTTP relay to an external app's `callback_url` (`kind=external`, Part 3).
- **Withdrawal claim/complete** (`withdrawal`): not UoW-based (both writes stay within the
  `withdrawal` module's own table) — instead uses an atomic conditional `UPDATE ... WHERE
  status=<expected>` (check rows-affected) so a concurrent double-claim/double-complete is
  impossible at the DB level, without needing a transaction across modules.
- **Tenant provisioning** (`platform`, via `bootstrap.ProvisionTenant`): store + default settings
  + first owner admin account, one transaction — the only way a new tenant is onboarded.

Each contract client is tx-aware: when invoked inside a UoW, its queries run on the shared
transaction (`uow.Q(ctx)`).

## Access-gate middleware (auth)

Beyond identity (`Authenticate`) and actor/role checks, `auth`'s middleware enforces two more
gates on **every** authenticated request, computed live (no caching):

- **Tenant suspension** (`stores.status`) — rejects `403` for `admin`/`staff` principals whose
  store is suspended. `platform`/`app` principals are exempt (no `store_id`).
- **Subscription gate** (`stores` via `subscriptionclient.Current`) — a store with no active
  package fully blocks `staff`, and restricts `admin` to an allow-listed set of routes, rejecting
  everything else `402 Payment Required` (distinct from `403` so the frontend can redirect
  instead of logging out).

Both checks require `auth` to depend on another module's contract client — wired via a
post-construction **setter**, not a constructor param, to avoid a circular dependency (`auth`
needs to exist before `subscription` can construct, but `subscription` needs `auth`'s middleware
to protect its own routes).

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
