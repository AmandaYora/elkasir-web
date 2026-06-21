# CLAUDE.md — Elkasir AI Gateway

Elkasir is a **multi-tenant POS for F&B**. One Go API serves a **React 19 web admin** dashboard and **customer self-order** pages; a separate Flutter app (`elkasir_pos`, not in this repo) is the cashier POS. This file is a concise gateway — **read the relevant `knowledge/` file before changing code.**

## Stack (locked)

- **Backend** `apps/api`: Go modular monolith — go-chi, MySQL via **sqlc + golang-migrate + database/sql** (no GORM), ULID ids, JWT auth.
- **Frontend** `apps/web`: **React 19 + Tailwind 4**, **react-router-dom** (lazy routes), **Zustand**, **Zod**, **Axios**, `@/*` alias. **No TanStack. No Turborepo.**
- **Packages**: `packages/api-contract` (OpenAPI → TS), `packages/shared` (domain-agnostic TS).
- **Deploy**: ONE container — SPA built statically and **embedded into the Go binary**; one process serves the SPA (root) + API (`/api/v1`). **MySQL runs at host/OS level**, never a DB container by default.

## Commands (run from repo root, web & api separately)

```bash
npm install                # install JS workspaces
npm run dev:api            # backend with Air live-reload (needs: go install github.com/air-verse/air@latest)
npm run dev:web            # frontend (Vite :8080)
npm run migrate:up         # apply DB migrations (golang-migrate)
npm run migrate:create -- <name>
npm run db:seed            # seed demo data
npm run sqlc:generate      # regenerate type-safe DB access
npm run gen:contract       # regenerate OpenAPI TS client
npm run build              # build web → embed into Go binary → build apps/api/bin/api (one container)
npm run docker:build && npm run docker:up   # one app container + host MySQL
```

API base path is **`/api/v1`**. Backend listens on `:8081`; web dev on `:8080`.

## Critical architecture rules

- **Modular monolith.** Each backend module lives under `apps/api/internal/modules/<module>/` with `contracts/`, `application/`, `domain/` (+`events/`), `infrastructure/`, `presentation/`, and a `<module>.module.go` wiring file. Only `contracts/` is public to other modules.
- **No cross-module** service/repository/domain imports, DB joins, or foreign keys. Cross-module relations are **primitive IDs**; cross-module lookups/flows go through the provider module's contract client (and the Unit-of-Work for atomic flows).
- **Multi-tenant**: every row is scoped by `store_id`, taken from the authenticated principal — never from the request body.
- **Frontend**: feature code lives in `apps/web/src/modules/<module>/` (pages/components/services/schemas/stores/hooks/types). Generic UI in `src/shared/`. All HTTP goes through `src/shared/services/http-client.ts` (Axios). Theme colors are centralized in `src/theme/`.

## Deployment — LIVE — build in CI, run on server

**Live at http://103.189.235.79** (deployed 2026-06-21). Production deploy is
**build-in-CI → GHCR → server pulls the image**. The VPS (2 GB RAM) **never compiles** — it
only pulls and runs. Full runbook + the exact live state (VPS facts, paths, what's
provisioned): **[docs/DEPLOYMENT_PIPELINE.md](docs/DEPLOYMENT_PIPELINE.md)**; basics in
[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

**Fast path to ship a change:** push to `main` → wait for CI green (the `image` job pushes
`ghcr.io/amandayora/elkasir-web:<git-sha>`) → on the VPS run `~/elkasir/deploy.sh <full-git-sha>`.
Rollback = same command with an older sha. That's it — nothing else to discover.

- **Flow**: push `main` → **`ci.yml` builds the image** (the `image` job, gated by
  `needs:[api,contract,web]`, push-to-main only) → pushes to **GHCR**
  (`ghcr.io/amandayora/elkasir-web:{latest,<git-sha>}`) → server **pulls** and runs it. Deploy
  & rollback are by image tag (`<full-git-sha>`), never a rebuild on the server. *(There is no
  separate `deploy.yml` — the build/push lives inside `ci.yml`.)*
- **One binary, four roles** (`apps/api/cmd/api`): `<none>`=serve · `migrate up|down [n]` ·
  `seed` · `healthcheck`. Migrations are `go:embed`-ed and run via golang-migrate-as-library,
  so the **same image** serves + migrates + seeds (distroless, no shell). The Dockerfile
  `HEALTHCHECK` calls `/app/api healthcheck`.
- **One container = the whole monorepo** (SPA embedded in the Go binary). **MySQL stays at
  host/OS level** (never a container); the container reaches it via `host.docker.internal`.
- **Host topology**: nginx (host) reverse-proxies `:80` → `127.0.0.1:8081` (TLS pending a
  domain — bare IP now); UFW opens `22/80/443` and `3306` only from the docker subnet
  (`172.16.0.0/12`, never public). **No IPv6 on this host** (no `listen [::]:80` in nginx).
- **Secrets** live only in `~/elkasir/.env` on the server (chmod 600) — never committed; the
  image holds no secrets (injected at runtime via `env_file`). GHCR pull creds are cached in
  the server's `~/.docker/config.json` (a `read:packages` PAT) — already logged in.
- **Migrations are forward-only**; keep them backward-compatible (expand→contract) so an older
  image still runs after a rollback.
- **Gate**: `ci.yml` (api/contract/web) must be green before the `image` job runs.
- **Don't**: build on the VPS, dockerize MySQL, expose `3306` publicly, push to `main`
  without CI green, or bake `.env` into the image.

> `npm run docker:build` / `docker:up` (build-based compose) is for **local one-container
> testing only** — not the VPS production path above.

## Knowledge base — read before editing

| Topic | File |
|---|---|
| Routing map for the knowledge base | [knowledge/INDEX.md](knowledge/INDEX.md) |
| What Elkasir is / why | [knowledge/PROJECT_BRIEF.md](knowledge/PROJECT_BRIEF.md) |
| Features & user stories | [knowledge/PRODUCT_REQUIREMENTS.md](knowledge/PRODUCT_REQUIREMENTS.md) |
| Architecture, boundaries, data flow | [knowledge/ARCHITECTURE_OVERVIEW.md](knowledge/ARCHITECTURE_OVERVIEW.md) |
| Each module + its public contract | [knowledge/MODULE_MAP.md](knowledge/MODULE_MAP.md) |
| Shared terminology | [knowledge/DOMAIN_GLOSSARY.md](knowledge/DOMAIN_GLOSSARY.md) |
| API conventions (`/api/v1`, envelopes, auth) | [knowledge/API_GUIDE.md](knowledge/API_GUIDE.md) |
| Tables, ownership, primitive-ID relations | [knowledge/DATABASE_GUIDE.md](knowledge/DATABASE_GUIDE.md) |
| Backend conventions | [knowledge/BACKEND_GUIDE.md](knowledge/BACKEND_GUIDE.md) |
| Frontend conventions | [knowledge/FRONTEND_GUIDE.md](knowledge/FRONTEND_GUIDE.md) |
| Decisions (ADRs) | [knowledge/decisions/](knowledge/decisions/) |

Path-scoped technical rules Claude must follow while editing are in [.claude/rules/](.claude/rules/). Product/architecture docs for humans are in [docs/](docs/).
