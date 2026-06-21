# Elkasir — Deployment & Operations

> How to set up, run, build, and deploy Elkasir. All `npm run …` commands are the
> **real scripts** from the root [`package.json`](../package.json) and are run from
> the **repo root**. For architecture see [`SYSTEM_DESIGN.md`](./SYSTEM_DESIGN.md).

---

## 1. Prerequisites

| Tool | Version | Required? | Notes |
|---|---|---|---|
| **Go** | 1.26 | Yes | Builds `apps/api` and the embedded one-container binary. |
| **Node.js** | 22 | Yes | Builds `apps/web` and runs the workspace scripts. |
| **npm** | 10.9.2 | Yes | Pinned via `packageManager`; manages the JS workspaces. |
| **MySQL** | 8 | Yes | Runs at **host/OS level** (not a container by default). |
| **Docker + Compose** | recent | Optional | Only for the one-container deploy (`docker:build` / `docker:up`). |
| **air** | latest | Dev only | Go live-reload watcher used by `npm run dev:api`. |

`sqlc` and `golang-migrate` do **not** need global installs — they run via
`go run …@version` inside the npm scripts.

Install the Air watcher once:

```bash
go install github.com/air-verse/air@latest
# ensure your Go bin dir (e.g. ~/go/bin or %USERPROFILE%\go\bin) is on PATH
```

---

## 2. Local setup (one time)

### 2.1 Environment files

`.env.example` (repo root) is the documented source of truth. Copy the relevant
blocks into the two app-level env files (the API loads its `.env` via godotenv;
the web app only exposes `VITE_*` variables to the browser):

```bash
cp .env.example apps/api/.env
cp .env.example apps/web/.env
```

For local dev, point the web app at the API:

```dotenv
# apps/web/.env
VITE_API_BASE_URL=http://localhost:8081/api/v1
```

(In the one-container production build the SPA is same-origin, so use
`VITE_API_BASE_URL=/api/v1`.)

### 2.2 Create the database

Create the database in your host MySQL (match `DB_NAME`, default `elkasir_db`):

```sql
CREATE DATABASE elkasir_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

The API builds its DSN from `DB_HOST` / `DB_PORT` / `DB_USERNAME` /
`DB_PASSWORD` / `DB_NAME` (defaults suit a local MySQL with user `root`, empty
password). Adjust `apps/api/.env` to match your MySQL.

### 2.3 Install, migrate, seed

```bash
npm install            # install JS workspaces (apps/web + packages/*)
npm run migrate:up     # apply DB migrations (golang-migrate via scripts/migrate.mjs)
npm run db:seed        # seed demo data (apps/api/cmd/seed)
```

Migration helpers:

```bash
npm run migrate:up                 # apply all pending migrations
npm run migrate:down               # roll back the last migration (down 1)
npm run migrate:create -- <name>   # create a new migration pair
```

---

## 3. Local run (two terminals)

Backend and frontend run as **two separate processes**, both started from the
repo root:

```bash
# Terminal 1 — backend (Go + Air live-reload), serves /api/v1 on :8081
npm run dev:api

# Terminal 2 — frontend (Vite dev server) on :8080
npm run dev:web
```

Default dev ports:

| Service | Port |
|---|---|
| Web (Vite dev) | 8080 |
| API (Go) | 8081 |
| MySQL | 3306 |

The backend needs MySQL up with migrations applied (and optionally seeded) before
it will start cleanly; otherwise it fails with `Access denied` / `connection
refused`.

### Codegen (when contracts/queries change)

```bash
npm run sqlc:generate   # regenerate type-safe DB access (sqlc)
npm run gen:contract    # regenerate the OpenAPI → TypeScript client
```

---

## 4. Production build (one binary)

`npm run build` produces a single self-contained binary that serves **both** the
SPA (root) and the API (`/api/v1`):

```bash
npm run build
```

This runs three steps in order:

1. `build:web` — Vite builds `apps/web` to static assets.
2. `scripts/embed-web.mjs` — copies the built SPA into the Go embed directory so
   `go:embed` bakes it into the binary.
3. `build:api` — compiles the Go binary to `apps/api/bin/api`.

Run the resulting binary with the production env in place (DB reachable, JWT
secret set):

```bash
./apps/api/bin/api
```

---

## 5. Docker (one app container + host MySQL)

The default deployment is **one app container**; MySQL stays at host/OS level and
is reached via `host.docker.internal`.

```bash
npm run docker:build   # docker build -t elkasir-app:latest -f infra/docker/Dockerfile .
npm run docker:up      # docker compose up -d --build
```

What the image does (`infra/docker/Dockerfile`, build context = repo root):

1. **Stage 1 (node:22-alpine):** `npm ci` then `npm run build -w @elkasir/web` →
   static SPA in `apps/web/dist`.
2. **Stage 2 (golang:1.26):** copies the SPA into `internal/webui/dist` and
   compiles the Go binary with `go:embed`.
3. **Stage 3 (distroless static, nonroot):** ships just the binary (and bundled
   migrations) and runs as the `nonroot` user.

`docker-compose.yml` runs one service, `app`:

- Maps `host.docker.internal → host-gateway` so the container reaches the host
  MySQL.
- Sets `DB_HOST=host.docker.internal`.
- Publishes only `127.0.0.1:8081:8081` — put a host reverse proxy
  (`infra/nginx/elkasir.conf`, or Caddy) in front to terminate TLS.

---

## 6. Host MySQL prerequisites (for the container)

Because the container connects to MySQL on the host, configure the host MySQL to
accept connections from the Docker network:

1. **Bind address** — MySQL must listen on the Docker bridge as well as
   localhost. In `my.cnf`, set a bind address that includes the bridge (e.g.
   `bind-address = 0.0.0.0`, or explicitly `127.0.0.1` **and** the docker bridge
   IP such as `172.17.0.1`).
2. **User grants for the Docker subnet** — the DB user must be allowed from the
   container subnet, e.g.:
   ```sql
   CREATE USER 'elkasir'@'172.%' IDENTIFIED BY '<strong-password>';
   GRANT ALL PRIVILEGES ON elkasir_db.* TO 'elkasir'@'172.%';
   FLUSH PRIVILEGES;
   ```
3. **Firewall** — allow inbound TCP `3306` from the Docker bridge network
   (`172.16.0.0/12`) on the host firewall, while keeping `3306` closed to the
   public internet.

Migrations are bundled in the image for tooling but are **not run on container
boot** — run `npm run migrate:up` against the host DB before/independent of
starting the app.

---

## 7. Environment variable reference

Copy from `.env.example`. Backend variables go in `apps/api/.env`; only
`VITE_*` reach the browser (`apps/web/.env`).

### App / server (`apps/api`)

| Variable | Example / default | Purpose |
|---|---|---|
| `API_ENV` | `development` / `production` | Runtime mode. |
| `API_ADDR` | `:8081` | Listen address. |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:8080,http://127.0.0.1:8080` | Comma-separated dev origins (usually empty in one-container prod — SPA is same-origin). |
| `PUBLIC_BASE_URL` | `http://localhost:8081` | Public base URL (payment callbacks/redirects). |

### Database (host MySQL 8)

| Variable | Default | Purpose |
|---|---|---|
| `DB_HOST` | `localhost` (`host.docker.internal` in Docker) | MySQL host. |
| `DB_PORT` | `3306` | MySQL port. |
| `DB_USERNAME` | `root` (`elkasir` in Docker) | DB user. |
| `DB_PASSWORD` | _(empty)_ — required in Docker | DB password. |
| `DB_NAME` | `elkasir_db` | Database name. |
| `DB_DSN` | _(optional)_ | Full go-sql-driver DSN override; built from `DB_*` if empty. |
| `DB_DSN_URL` | _(optional)_ | golang-migrate URL override used by `scripts/migrate.mjs`. |

### JWT

| Variable | Default | Purpose |
|---|---|---|
| `JWT_SECRET` | _(required, ≥ 32 chars)_ | Token signing secret. |
| `JWT_ACCESS_TTL` | `15m` | Access-token lifetime. |
| `JWT_REFRESH_TTL` | `168h` | Refresh-token lifetime. |

### Payments — QRIS (provider-agnostic: Tripay or Midtrans)

One provider is active, chosen by `PAYMENT_PROVIDER`; `PAYMENT_ENV` picks sandbox vs production
(each provider's `*_BASE_URL` is derived from it unless overridden).

| Variable | Default | Purpose |
|---|---|---|
| `PAYMENT_PROVIDER` | _(empty = simulation)_ | `tripay` \| `midtrans`. Selects the active gateway. |
| `PAYMENT_ENV` | `sandbox` | `sandbox` \| `production` — derives each provider's base URL. |
| `TRIPAY_API_KEY` | _(empty)_ | Tripay API Key (Bearer auth for charge). |
| `TRIPAY_PRIVATE_KEY` | _(empty)_ | Tripay Private Key (charge signature + callback verification). |
| `TRIPAY_MERCHANT_CODE` | _(empty)_ | Tripay Merchant Code (part of the charge signature). |
| `TRIPAY_QRIS_METHOD` | `QRIS` | Tripay channel code for QRIS. |
| `TRIPAY_BASE_URL` | _(derived)_ | Override; default `…/api-sandbox` (sandbox) or `…/api` (production). |
| `MIDTRANS_SERVER_KEY` | _(empty)_ | Midtrans Server Key (Basic Auth charge + webhook signature). |
| `MIDTRANS_BASE_URL` | _(derived)_ | Override; default `api.sandbox.midtrans.com` or `api.midtrans.com`. |

Empty credentials for the active provider → QRIS runs in **simulation** (dev). Webhook endpoint
to register in the active provider's dashboard: `<PUBLIC_BASE_URL>/api/v1/webhooks/payment`.
Authenticity is verified per provider (Tripay `X-Callback-Signature` HMAC-SHA256 of raw body;
Midtrans `signature_key` SHA512).

### Frontend (`apps/web`)

| Variable | Value | Purpose |
|---|---|---|
| `VITE_API_BASE_URL` | dev: `http://localhost:8081/api/v1` · prod: `/api/v1` | Base URL the Axios http-client targets. |

---

## 8. Health checks

| Endpoint | Purpose |
|---|---|
| `GET /healthz` | Liveness — the process is up. |
| `GET /readyz` | Readiness — the database is reachable. |

Both live at the **root** (outside `/api/v1`) so infrastructure probes and the
container healthcheck can hit them directly.
