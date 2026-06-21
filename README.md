# Elkasir — Monorepo POS F&B

Monorepo for **Elkasir**, a multi-tenant point-of-sale for F&B businesses:

- **`apps/web`** — React 19 + Tailwind 4 admin dashboard + customer self-order pages (Vite SPA).
- **`apps/api`** — Go **modular monolith** (go-chi, MySQL via sqlc + golang-migrate, JWT).
- **`packages/api-contract`** — OpenAPI spec → generated TS client.
- **`packages/shared`** — domain-agnostic TS shared across apps.

Built with **plain npm workspaces — no Turborepo**, and a **react-router-dom + Zustand + Axios** frontend — **no TanStack**. For release, the SPA is built statically and **embedded into the Go binary** → **one container** serves the web SPA (root) and the API (`/api/v1`). A separate Flutter app `elkasir_pos` (not in this repo) is the cashier POS and only consumes the API.

> AI agents & new devs: start at [CLAUDE.md](./CLAUDE.md) and [knowledge/INDEX.md](./knowledge/INDEX.md).
> Human docs: [docs/](./docs/) (PRD, SYSTEM_DESIGN, API_CONTRACT, DB_SCHEMA, DEPLOYMENT).

## Structure

```
apps/
  web/          # React 19 SPA: src/{app,modules,shared,theme,styles}
  api/          # Go modular monolith: internal/{app,modules/<m>/{contracts,application,domain,infrastructure,presentation},platform,webui}
packages/
  api-contract/ # openapi.yaml → generated/ts client
  shared/       # domain-agnostic TS
docs/           # PRD, SYSTEM_DESIGN, API_CONTRACT, DB_SCHEMA, DEPLOYMENT
knowledge/      # project knowledge base (AI gateway)
.claude/rules/  # path-scoped editing rules
infra/          # docker/Dockerfile (one-container) + nginx/ (optional reverse proxy)
scripts/        # cross-platform helpers (embed-web, migrate)
docker-compose.yml  # one app container + host MySQL
```

## Prerequisites

| Tool | For | Required |
| --- | --- | --- |
| Node ≥ 20 + npm | web & contract codegen | yes |
| Go ≥ 1.26 | API | yes |
| MySQL 8 (host/Laragon) | database | yes |
| Air (`go install github.com/air-verse/air@latest`) | API live-reload | yes (for `dev:api`) |
| Docker + Compose | container release | optional |

> `sqlc` & `golang-migrate` need no global install — they run via `go run pkg@version` from the npm scripts.

## Local development (no Docker) — two terminals from the repo root

```bash
# once: env + database
cp .env.example apps/api/.env       # adjust DB_* to your MySQL (Laragon: user root, empty pass)
cp .env.example apps/web/.env       # keep VITE_API_BASE_URL=http://localhost:8081/api/v1
# create the database to match DB_NAME (e.g. elkasir_db), then:
npm install
npm run migrate:up
npm run db:seed                     # demo admin: adi@elkasir.id / admin123 (see seed output)

# run (two terminals, both from root)
npm run dev:api                     # Go API on http://localhost:8081 (REST under /api/v1)
npm run dev:web                     # web SPA on http://localhost:8080
```

## Root commands

| Command | Description |
| --- | --- |
| `npm run dev:api` / `npm run dev:web` | start backend (Air) / frontend (Vite), separately |
| `npm run migrate:up` / `migrate:down` / `migrate:create -- <name>` | golang-migrate |
| `npm run db:seed` | seed demo data |
| `npm run sqlc:generate` | regenerate type-safe DB access |
| `npm run gen:contract` | regenerate OpenAPI TS client |
| `npm run build` | build web → embed into Go binary → `apps/api/bin/api` (one container) |
| `npm run docker:build` / `docker:up` | build & run one app container (host MySQL) |

## Build & deploy (one container)

The SPA is built statically and embedded into the Go binary, so a single process serves web + API.

```bash
npm run build                      # local binary with embedded SPA → apps/api/bin/api
npm run docker:build               # image elkasir-app:latest (infra/docker/Dockerfile)
npm run docker:up                  # one container; MySQL stays on the HOST (host.docker.internal)
```

Expose publicly via a host reverse proxy (Caddy/nginx) + TLS — see [docs/DEPLOYMENT.md](./docs/DEPLOYMENT.md) and [infra/nginx/elkasir.conf](./infra/nginx/elkasir.conf).

## Ports

| Service | Port |
| --- | --- |
| Web (Vite dev) | 8080 |
| API (Go) | 8081 |
| MySQL | 3306 |
