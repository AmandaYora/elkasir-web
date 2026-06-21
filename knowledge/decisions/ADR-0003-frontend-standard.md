# ADR-0003: Frontend standard

- **Status:** Accepted
- **Date:** 2026-06-18
- **Deciders:** Elkasir engineering

## Context

`apps/web` is a single React SPA that delivers two surfaces — the **admin dashboard**
(owners/managers) and the **customer self-order pages** — and is, at release, embedded
into the Go API binary (see ADR-0001).

The existing codebase had accreted a heavy frontend stack: **TanStack Router** for
routing, **TanStack Query** for server state, and a **Radix/shadcn** component kit. For
a CRUD-and-forms admin app plus a small self-order flow, this proved to be more
machinery than the problem warranted:

- TanStack Router's typed-route generation and TanStack Query's caching/invalidation
  layer add concepts, configuration, and bundle weight that a straightforward
  request/response admin app does not need.
- The Radix/shadcn kit pulled in a large surface of primitives and generated component
  files, most of which we use thinly or could express more simply.

The team has adopted an **anti-overengineering standard**: prefer the smallest stack
that does the job clearly, and own the small amount of UI we actually use.

## Decision

Standardize `apps/web` on the following minimal stack:

| Concern | Choice |
| --- | --- |
| UI library | **React 19** |
| Styling | **Tailwind 4** |
| Routing | **react-router-dom**, with **lazy-loaded routes** (code-split per route) |
| Client state | **Zustand** |
| Validation | **Zod** (form/input and API-boundary validation) |
| HTTP | **Axios** |
| Import alias | **`@/*`** mapped to `src/*` |

### Structure

Organize the app by **module**, mirroring the backend's capability split rather than by
technical file type:

```
src/
  app/      # app shell, providers, router setup, layout
  modules/  # feature modules (products, tables, transactions, shifts, ...)
  shared/   # cross-module UI primitives, hooks, utilities, the minimal UI kit
  theme/    # centralized theme (tokens, colors, typography) — single source of truth
  styles/   # global stylesheet / Tailwind entry
```

### Explicit removals and rebuild

1. **Remove TanStack Router.** Replace it with **react-router-dom** using lazy routes.
2. **Remove TanStack Query.** Server interaction is plain **Axios** calls; any
   client-side state that needs sharing lives in **Zustand**. We accept manual data
   fetching/refresh in exchange for far fewer moving parts.
3. **Drop the Radix/shadcn kit and rebuild a minimal in-house UI kit** under
   `shared/`, containing only the components Elkasir actually uses, styled with
   Tailwind 4 and driven by the centralized `theme/`.

### Conventions
- **Lazy routes** keep the initial bundle small — important since the SPA ships inside
  the API binary and serves both admin and self-order audiences.
- **Centralized theme.** Design tokens live in `theme/`; components consume tokens
  rather than hard-coding values, so look-and-feel changes in one place.
- **Zod at boundaries.** Validate form input and parse/validate data crossing the API
  boundary, matching the OpenAPI contract types from `packages/api-contract`.

## Consequences

### Positive
- **Smaller, simpler stack.** Fewer dependencies and concepts; new contributors learn
  react-router-dom + Zustand + Axios + Zod rather than two TanStack subsystems plus a
  large component library.
- **Smaller bundle.** Dropping TanStack Query and the full Radix/shadcn surface, plus
  per-route lazy loading, reduces shipped JavaScript — valuable for an embedded SPA.
- **Full control of the UI kit.** Owning a minimal component set under `shared/` means
  no fighting a third-party kit's abstractions; styling flows from one theme.
- **Module-based structure** mirrors the backend, making the full-stack mental model
  consistent.

### Negative / trade-offs
- **We give up TanStack Query's caching, dedup, and background refetch.** Data
  freshness and re-fetching are now explicit work in our code; we accept this for
  simplicity, and can introduce a small fetching helper if needed.
- **Building/maintaining our own UI kit is ongoing work.** We must implement
  accessibility and edge cases that Radix handled for us; mitigated by keeping the kit
  small and only adding components when actually needed.
- **Migration cost.** Existing routes, queries, and shadcn components must be rewritten
  onto the new standard.
- **Less typed routing.** react-router-dom does not give TanStack Router's generated
  route types; route correctness leans more on convention and review.

### Rationale summary
The driving principle is **anti-overengineering**: choose the minimal stack that
clearly solves an admin-CRUD-plus-self-order app, own the little UI we need, and avoid
carrying framework machinery whose benefits we were not using. This aligns the frontend
with the same "smallest thing that works" philosophy applied to the monorepo (ADR-0001)
and the backend modular monolith (ADR-0002).
