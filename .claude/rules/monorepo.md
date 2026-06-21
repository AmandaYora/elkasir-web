# Rule: Monorepo (applies repo-wide)

- Structure is locked: `apps/web` (React 19 + Tailwind 4) + `apps/api` (Go modular monolith) +
  `packages/api-contract` + `packages/shared`. Do not add new top-level app/package dirs without an ADR.
- Build orchestration is **plain npm workspaces**. **Do NOT introduce Turborepo, Nx, or `concurrently`**
  as the default workflow. Frontend and backend start separately: `npm run dev:web`, `npm run dev:api`.
- **Do NOT introduce** (unless the user explicitly asks): Microservices, Kubernetes, Redis/Memcache,
  TanStack libraries, a Dockerized database, multiple frontend containers, separate web/api prod
  containers, cross-module foreign keys, cross-module joins, GORM.
- Deployment default is **one app container** (SPA embedded in the Go binary serving `/api/v1` + SPA)
  with **MySQL at host level**. Keep it that way.
- Secrets live only in `.env` (gitignored). Keep `.env.example` current; never commit real secrets.
- Cross-platform: helper scripts in `scripts/` must be Node `.mjs` (Windows + Unix). Avoid bash-only
  build steps in npm scripts.
