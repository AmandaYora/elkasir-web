# ADR-0001: Monorepo structure

- **Status:** Accepted
- **Date:** 2026-06-18
- **Deciders:** Elkasir engineering

## Context

Elkasir is one product made of several deliverables that must stay in lockstep:

- a **web admin dashboard** and **customer self-order pages** (one React 19 SPA),
- a **Go API** that serves both the SPA and the REST endpoints,
- a **separate Flutter cashier app** (`elkasir_pos`) that consumes the same API, and
- a **shared API contract** that defines the wire types both web and mobile rely on.

These pieces share types and conventions and are released together (the SPA is even
embedded into the API binary). Splitting them across multiple repositories would make
contract changes painful: a single field rename would require coordinated commits,
version bumps, and PR choreography across repos, with constant drift between the API's
DTOs and the frontend's types.

We also want to avoid operational and tooling overhead. The team is small and the
deploy target is intentionally simple (one container, host-level database). Heavy
build orchestration would add cost without buying us much at this scale.

## Decision

Adopt a **single monorepo** with the following top-level layout:

```
apps/
  web/    # React 19 + Tailwind 4 SPA: admin dashboard + self-order pages
  api/    # Go modular monolith; serves the SPA at root and the API under /api/v1
packages/
  api-contract/   # OpenAPI contract — the source of truth for wire types
  shared/         # domain-agnostic TypeScript utilities shared across web packages
knowledge/        # project documentation (briefs, requirements, glossary, ADRs)
```

Concrete decisions:

1. **`apps/` for deployables, `packages/` for shared libraries.** `apps/web` and
   `apps/api` are the two things we run; `packages/*` are consumed by them.
2. **npm workspaces** manage JavaScript/TypeScript dependencies and linking across
   `apps/web` and `packages/*`. The Go API uses Go modules; npm workspaces do not
   manage it but it lives in the same repo.
3. **No Turborepo (and no other monorepo build orchestrator).** Tasks are plain npm
   scripts run from the repo root — notably `dev:web` (runs the Vite SPA dev server)
   and `dev:api` (runs the Go API). Two terminals, two commands, no remote-cache
   layer, no task graph to configure.
4. **OpenAPI as the contract source of truth.** `packages/api-contract` holds the
   OpenAPI document; generated TypeScript types flow from it to the web app (and are
   the reference for the Flutter app), keeping API and clients aligned.
5. **One-container deploy with a host-level database.** For release, the SPA is built
   to static assets and embedded into the Go binary, so **one container** serves both
   the web app (at root) and the API (under `/api/v1`). **MySQL runs at the host/OS
   level**, not in a container; the app container connects to it over the host network.
   Public exposure is via a host reverse proxy (Caddy/nginx) + TLS.

## Consequences

### Positive
- **Atomic cross-cutting changes.** A contract change and its API + web consumers land
  in one commit/PR; no cross-repo version dance.
- **Single source of truth for types** via the OpenAPI contract reduces drift between
  backend and frontend.
- **Low cognitive and tooling overhead.** Plain npm scripts (`dev:web`, `dev:api`) are
  trivial to understand; there is no build-orchestration config to learn or maintain.
- **Dead-simple deployment.** One artifact (one binary → one container) plus a host
  database. Fewer moving parts means fewer failure modes and easier ops for a small
  team.

### Negative / trade-offs
- **No build caching or task graph.** Without Turborepo, large repeated builds are not
  cached and cross-task ordering is manual. Acceptable at current scale; revisit if
  build times or task fan-out grow.
- **Mixed-ecosystem repo.** Go and Node/TypeScript coexist; contributors must have both
  toolchains. npm workspaces only manage the JS side.
- **The Flutter app lives outside this repo.** It cannot take an in-repo dependency on
  the contract package; it must consume the published/generated contract artifacts,
  so contract changes still need a deliberate sync step for mobile.
- **Coupled release cadence.** Embedding the SPA in the API binary means web and API
  ship together; you cannot release one without rebuilding the other.

### Notes
- The one-container model and host-level DB are described further in
  `knowledge/PROJECT_BRIEF.md`.
- The internal structure of `apps/api` is covered by ADR-0002 (modular monolith) and
  of `apps/web` by ADR-0003 (frontend standard).
