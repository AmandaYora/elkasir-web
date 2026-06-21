# Elkasir — Production Deploy Pipeline (CI → GHCR → server pull)

> **Status: design/blueprint.** This documents the *agreed* deploy flow (Option A).
> Not all of it is wired up yet — it's the standard to implement and follow.
> For local dev & the build-on-machine basics, see [`DEPLOYMENT.md`](./DEPLOYMENT.md).

## 0. TL;DR — the golden rule

**Build somewhere powerful, run somewhere small.** The VPS (2 vCPU / **2 GB RAM**)
must **never compile** the app (Vite + Go multi-stage build needs ~1–1.5 GB and would
risk OOM and downtime). Instead:

```
 ┌─ local ─┐        ┌──── GitHub ────┐      ┌─ GitHub Actions ─┐     ┌─── GHCR ───┐
 │  code   │ push   │   main branch  │ ───► │  CI gate + build │ ──► │  registry  │
 └─────────┘ ─────► └────────────────┘      └──────────────────┘     └─────┬──────┘
                                                                           │ pull (no build)
                                                                     ┌─────▼──────────────────┐
                                                                     │  VPS 103.189.235.79     │
                                                                     │  nginx :80/:443  ─proxy►│  ┌──────────────┐
                                                                     │  app container :8081    │  │ host MySQL   │
                                                                     │  (1 monorepo = 1 image) │──► 127.0.0.1:3306│
                                                                     └─────────────────────────┘  └──────────────┘
```

This keeps the mental model ("push to `main`, server gets it") — the only change is
the **server pulls a ready-made image**, not source code to build.

> **Branch:** production branch is **`main`** (matches `.github/workflows/ci.yml`). Do not
> deploy from any other branch.

---

## 1. Architecture (matches the project's locked design)

- **1 container = the whole monorepo.** The SPA (`apps/web`) is built and **embedded
  into the Go binary** (`apps/api`); one process serves the SPA at `/` and the API at
  `/api/v1`. Runtime image is **distroless/static, nonroot** (tiny, no shell).
- **MySQL stays at host/OS level** (already provisioned), reached from the container via
  `host.docker.internal` (Docker host-gateway). Never a DB container.
- **nginx on the host** is the public reverse proxy (TLS), forwarding `:80/:443` →
  `127.0.0.1:8081` (the app publishes only to localhost).
- **GHCR** (GitHub Container Registry, **private**) stores versioned, immutable images.
  Rollback = re-point to a previous tag — no rebuild.

---

## 2. What's already provisioned on the VPS (done)

| Item | State |
|---|---|
| Docker Engine + Compose plugin | installed (29.x / v5.x) |
| MySQL 8 (host) | `elkasir_db` + user `elkasir_user@'172.%'` (`mysql_native_password`, rights on `elkasir_db` only) |
| Host MySQL reachable from containers | verified via `host.docker.internal`, port 3306 |
| Firewall (UFW) | `22/80/443` open; `3306` only from `172.16.0.0/12` (closed to public) |
| `~/elkasir/.env` | production env (DB creds + `JWT_SECRET`), `chmod 600`, never committed |
| nginx (host) | running, no sites yet (ready to host the reverse-proxy config) |

> **DB auth plugin (verified):** `elkasir_user` uses **`mysql_native_password`**, not MySQL 8's
> default `caching_sha2_password`. The app's DSN connects over **plaintext** on the host-internal
> docker bridge without TLS/`allowPublicKeyRetrieval`; `caching_sha2_password` rejects that first
> auth (`ERROR 2061: Authentication requires secure connection`). `native_password` is safe here
> (loopback — traffic never leaves the host) and works with the app's stock DSN. *Alternative if
> you prefer the modern plugin:* keep `caching_sha2_password` and set
> `DB_DSN=...&allowPublicKeyRetrieval=true` in `~/elkasir/.env`. Verified: a plaintext container
> connection (mirroring the Go driver) authenticates and sees only `elkasir_db`.

Remaining work: **CI to build/push the image** + **server files to pull/run** +
**migration/seed/healthcheck subcommands** (§6/§9) + **nginx TLS** (§10).

---

## 3. Image naming & tags

GHCR requires lowercase. Owner `AmandaYora` → `amandayora`.

| Image | Purpose |
|---|---|
| `ghcr.io/amandayora/elkasir-web:<git-sha>` | the app (immutable, per-commit) |
| `ghcr.io/amandayora/elkasir-web:latest` | convenience pointer to newest `main` |

Always deploy by **`<git-sha>`** (or a `vX.Y.Z` tag), not `latest`, so rollback is exact.
The image contains **no secrets** (`.env` is gitignored **and** excluded by
`.dockerignore` via `**/.env` — verified); secrets are injected at runtime via `env_file`.
The single image also runs migrations/seed/healthcheck via subcommands (§9), so **no
second image is needed**.

---

## 4. CI — GitHub Actions

### 4a. Gate first (existing `ci.yml`)
`.github/workflows/ci.yml` already runs on `push: [main]` + all PRs: Go `vet/build/test`,
OpenAPI↔TS sync check, and web `lint/build`. **This must stay green** — it is the gate.

### 4b. Build & push image (`.github/workflows/deploy.yml`)
On push to `main`, after/with CI: build & push the app image to GHCR.

```yaml
name: build-and-push
on:
  push:
    branches: [main]
permissions:
  contents: read
  packages: write          # push to GHCR
concurrency:
  group: deploy-${{ github.ref }}
  cancel-in-progress: true
jobs:
  image:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}   # built-in, no PAT needed
      - uses: docker/build-push-action@v6
        with:
          context: .
          file: infra/docker/Dockerfile
          push: true
          tags: |
            ghcr.io/amandayora/elkasir-web:latest
            ghcr.io/amandayora/elkasir-web:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

> Optionally make this job `needs:` the CI jobs (or merge into `ci.yml`) so an image is
> only built when lint/test pass. Never push straight to `main` expecting a deploy without
> green CI.

---

## 5. GHCR access from the server (image is **private**)

Keep the package **private** — the image is your compiled product (Go binary + SPA); don't
publish it. The server logs in once with a **read-only** token:

```bash
echo "<GHCR_READ_PAT>" | docker login ghcr.io -u <github-user> --password-stdin
```

`<GHCR_READ_PAT>` = a fine-grained PAT with **`read:packages`** only (store nowhere else).
Docker caches the credential so `compose pull` works unattended afterwards.

---

## 6. Database migrations (run via the binary — single artifact)

The runtime image is **distroless** (no `go`, `node`, shell) so `npm run migrate:up` can't
run there. **Agreed approach: a `migrate` subcommand on the binary** (golang-migrate as a
*library*, migrations via `go:embed`). Then the **same image** that serves also migrates:

```bash
docker run --rm --add-host=host.docker.internal:host-gateway --env-file ~/elkasir/.env \
  ghcr.io/amandayora/elkasir-web:<tag> migrate up
```

This needs the subcommand implemented (§9) — small, high value: one immutable artifact does
serve + migrate + seed + healthcheck, nothing else to build or version.

**Interim fallback (until the subcommand lands), no app code change:** a dedicated
`migrate` image — `infra/docker/Dockerfile.migrate`:
```dockerfile
FROM migrate/migrate:v4.18.1
COPY apps/api/db/migrations /migrations
ENTRYPOINT ["migrate"]
```
run as `... elkasir-web-migrate:<tag> -path=/migrations -database "mysql://…" up`. Prefer the
subcommand; retire this image once it exists.

> **Migration discipline:** migrations are **forward-only**. Code rolls back instantly by
> re-pointing to an old image, but the schema does not — always write **backward-compatible
> (expand→contract)** migrations so the previous image still runs against the new schema.
>
> **Seeding** the first store + admin is a one-time bootstrap (`/app/api seed` once via §9),
> then never again.

---

## 7. Server files & manual deploy runbook

Two files live in `~/elkasir/` on the VPS (`.env` already exists):

### `~/elkasir/docker-compose.prod.yml` (image-only — **no `build:`**)

```yaml
services:
  app:
    image: ghcr.io/amandayora/elkasir-web:${IMAGE_TAG:-latest}
    container_name: elkasir-app
    restart: unless-stopped
    env_file: .env
    extra_hosts:
      - "host.docker.internal:host-gateway"   # reach host MySQL
    ports:
      - "127.0.0.1:8081:8081"                  # localhost only; nginx fronts it
    healthcheck:
      test: ["CMD", "/app/api", "healthcheck"] # needs the subcommand (§9)
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 20s
    logging:                                   # cap logs so disk can't fill
      driver: json-file
      options: { max-size: "10m", max-file: "3" }
```

### `~/elkasir/deploy.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
TAG="${1:-latest}"; export IMAGE_TAG="$TAG"

echo "▶ pull image ($TAG)"
docker compose -f docker-compose.prod.yml pull

echo "▶ run migrations (same image, subcommand)"
docker run --rm --add-host=host.docker.internal:host-gateway --env-file .env \
  "ghcr.io/amandayora/elkasir-web:$TAG" migrate up
# Interim (no subcommand yet): use the elkasir-web-migrate image instead (see §6).

echo "▶ start app"
docker compose -f docker-compose.prod.yml up -d

echo "▶ verify"
sleep 3; curl -fsS http://127.0.0.1:8081/readyz && echo "  OK"
docker image prune -f >/dev/null
```

**Deploy = one command** on the server (or via §8 from CI):

```bash
~/elkasir/deploy.sh <git-sha>     # use latest only for quick tests
```

### Rollback (instant, no rebuild)

```bash
~/elkasir/deploy.sh <previous-git-sha>     # re-points to an older image (forward-only DB; see §6)
```

> **Downtime:** `compose up -d` recreates the single container → a **few-seconds blip**, not
> zero-downtime. Acceptable at this scale. If true zero-downtime is needed later, run two
> replicas behind nginx and switch — not required now.

---

## 8. Optional: push-to-deploy (CI runs the deploy over SSH)

To make "push to `main`" actually deploy, add a `deploy` job that SSHes to the VPS after the
image is pushed, using a **dedicated CI deploy key** (separate from personal keys):

1. On the VPS: create a key, add its **public** part to `~/.ssh/authorized_keys`.
2. GitHub repo **Settings → Secrets**: `DEPLOY_HOST=103.189.235.79`,
   `DEPLOY_USER=dimasprasetio`, `DEPLOY_SSH_KEY=<private key>`.
3. Append to the workflow:
```yaml
  deploy:
    needs: image
    runs-on: ubuntu-latest
    steps:
      - uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.DEPLOY_HOST }}
          username: ${{ secrets.DEPLOY_USER }}
          key: ${{ secrets.DEPLOY_SSH_KEY }}
          script: ~/elkasir/deploy.sh ${{ github.sha }}
```
Prefer a **manual** `deploy.sh` run first (a human gate); enable auto-deploy once trusted.

---

## 9. Binary subcommands (the single-artifact enabler)

Because the runtime is distroless, teach `cmd/api` to branch on `argv[1]` so the one image
is self-sufficient:

| Command | Does |
|---|---|
| `/app/api` | serve (default) |
| `/app/api migrate up` | apply `go:embed`-ed migrations (replaces the migrate image) |
| `/app/api seed` | one-time bootstrap (store + admin) |
| `/app/api healthcheck` | `GET localhost:8081/readyz`, exit 0/1 (Docker HEALTHCHECK) |

This collapses migrate-image + seed + healthcheck into the artifact you already ship — the
cleanest, most reproducible shape. Until it lands, use the §6 interim image and drop the
compose `healthcheck`.

---

## 10. nginx reverse proxy + TLS (host)

The app listens on `127.0.0.1:8081`; nginx terminates TLS and proxies. Starter config:
[`infra/nginx/elkasir.conf`](../infra/nginx/elkasir.conf). Enable it:

```bash
sudo cp infra/nginx/elkasir.conf /etc/nginx/sites-available/elkasir   # set server_name first
sudo ln -s /etc/nginx/sites-available/elkasir /etc/nginx/sites-enabled/elkasir
sudo nginx -t && sudo systemctl reload nginx
# TLS once a domain points here:
sudo apt-get install -y certbot python3-certbot-nginx
sudo certbot --nginx -d <your-domain>      # auto-renews via systemd timer
```

After TLS, set `PUBLIC_BASE_URL=https://<domain>` (and `CORS_ALLOWED_ORIGINS` if the Flutter
app calls cross-origin) in `~/elkasir/.env`, then re-run `deploy.sh`.

---

## 11. Backups & restore (host MySQL)

MySQL is on the host → easy nightly dumps. Use a credentials file so no password on the
command line:

```bash
# ~/.my.cnf  (chmod 600)  →  [client] user=elkasir_user  password=...
0 2 * * *  mysqldump --single-transaction --routines elkasir_db | gzip > ~/backups/elkasir_$(date +\%F).sql.gz
30 2 * * * find ~/backups -name 'elkasir_*.sql.gz' -mtime +14 -delete   # keep 14 days
```

**Restore** (test this periodically — an untested backup is not a backup):

```bash
gunzip < ~/backups/elkasir_<date>.sql.gz | mysql elkasir_db
# fresh box: CREATE DATABASE elkasir_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; then import.
```

(The previous project's backup remains at `~/backups/stovo_<ts>/` until you remove it.)

---

## 12. Observability (lightweight, no extra infra)

- **Logs**: `docker logs -f elkasir-app` (capped via the compose `logging` block above).
- **Liveness/readiness**: `GET /healthz` (process) and `GET /readyz` (DB). Wire `/readyz`
  into the compose healthcheck (§9) and, optionally, an external uptime monitor.
- **Resources**: `docker stats elkasir-app` and host `free -h` / `df -h` — the box is 2 GB,
  watch memory after deploys.
- **Restart policy**: `restart: unless-stopped` recovers the app across crashes/reboots.

No log stack / metrics server is warranted at this scale (consistent with the project's
anti-over-engineering rules); add one only if traffic justifies it.

---

## 13. First-deploy checklist (when you say "go")

1. Implement §9 subcommands (`migrate`/`seed`/`healthcheck`) — or use the §6 interim image.
2. Add `.github/workflows/deploy.yml`; ensure `ci.yml` is green on `main`.
3. Push to `main` → confirm Actions built & pushed the image to GHCR (private).
4. On the server: `docker login ghcr.io` (read-only PAT).
5. Copy `docker-compose.prod.yml` + `deploy.sh` to `~/elkasir/` (`chmod +x deploy.sh`).
6. `~/elkasir/deploy.sh <git-sha>` → migrations run, app starts, `/readyz` OK.
7. One-time: `/app/api seed`, then log in and change the admin password.
8. Add the nginx site + TLS (§10).
9. Enable nightly backups + verify a restore (§11).
10. (Optional) turn on push-to-deploy (§8).

---

## 14. Why this is "healthy" vs build-on-server

| Concern | build-on-server (`git pull && up --build`) | this pipeline (pull image) |
|---|---|---|
| 2 GB RAM build | ⚠️ OOM risk, competes with live app | ✅ builds on Actions, server only pulls |
| Rollback | rebuild old commit (slow/fragile) | ✅ re-point to old tag, instant |
| Downtime | during whole build | ✅ only the short container swap |
| Reproducibility | depends on server toolchain/state | ✅ immutable, identical image everywhere |
| Secrets | source + toolchain on server | ✅ only `.env` + image on server (image has none) |
| Gate before prod | none | ✅ CI must be green on `main` first |
