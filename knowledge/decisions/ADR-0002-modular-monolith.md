# ADR-0002: Go modular monolith with strict module boundaries

- **Status:** Accepted
- **Date:** 2026-06-18
- **Deciders:** Elkasir engineering

## Context

The Elkasir backend covers a fair number of business areas: authentication, admin
users, POS staff, products, categories, tables, transactions, shifts, cash movements,
withdrawals, reports, self-orders, and payments. These areas are related but should not
become a tangled ball of mutual imports where every package reaches into every other
package's internals and database tables.

At the same time, Elkasir is operated by a small team and deploys as **one container**
(see ADR-0001). The traffic and team size do not justify the operational weight of
microservices — separate deployables, network hops, distributed transactions, and
service discovery — just to get clean boundaries.

We want the **internal decoupling benefits** that people reach for microservices to
get, without paying the **distributed-systems tax**.

## Decision

Build the API as a **modular monolith**: one Go process and one deployable, internally
partitioned into modules with **strictly enforced boundaries**.

### Module layout

Each business capability is a module under `internal/modules/<module>/` with a fixed
set of sub-packages:

- `contracts/` — the module's **only public surface**: a client interface, the DTOs it
  exchanges, and sentinel errors.
- `application/` — use cases / services (orchestration of the module's own work).
- `domain/` — entities, value objects, and `events/`.
- `infrastructure/` — repositories, sqlc data access, and the implementation of the
  module's `contracts/` client.
- `presentation/` — HTTP handlers and route registration.
- `<module>.module.go` — the wiring that assembles the module.

Shared technical (non-business) code lives in `internal/platform/`: `config`,
`db`/`sqlcgen`, `httpserver`, `httpx`, `id`, and `uow` (Unit of Work).

### Boundary rules (the locked standard)

1. **Only `contracts/` is public.** No module may import another module's
   `application/`, `domain/`, `infrastructure/`, or `presentation/` packages. Cross-
   module calls go exclusively through the other module's `contracts/` client
   interface.
2. **No cross-module service or repository imports.** A module never instantiates or
   reaches into another module's services or repos directly.
3. **No cross-module foreign keys or joins.** A module owns its own tables. It must not
   define an FK to, or SQL-join against, another module's tables.
4. **Primitive-ID relations.** References across modules are stored as plain ID values
   (e.g. a transaction holds a `productId`, `shiftId`, `tableId`), not as object
   references or DB relationships. Hydration, when needed, happens by calling the owning
   module's contract client.
5. **Cross-module flows via contract clients + Unit of Work.** Multi-module operations
   are orchestrated by one module's service calling the contract clients of the others,
   all running inside a shared **Unit of Work** so the whole flow commits or rolls back
   atomically.

The composition root wires modules together by injecting **contract clients**, not
concrete implementations. For example, recording a sale is orchestrated by the
transaction service using the product, shift, and sales contract clients under one UoW;
the self-order flow orchestrates product, sales, shift, table, and payment clients the
same way. This keeps every cross-module dependency explicit and visible at the seams.

## Consequences

### Positive
- **Enforced decoupling without distribution.** Modules talk only through narrow
  contract interfaces, so internals can evolve freely behind them — yet it is still one
  process, one binary, one container.
- **Local, easy reasoning about transactions.** Because everything shares one database
  process, multi-module atomicity is achieved with an in-process Unit of Work rather
  than sagas or two-phase commit.
- **Clear ownership.** Each module owns its tables and its public contract; "who can
  change this?" has an obvious answer.
- **A viable extraction path.** If a module ever truly needs to become its own service,
  its already-defined `contracts/` interface is the seam to split along — the refactor
  is bounded.
- **Faster onboarding.** The same sub-package shape (`contracts` / `application` /
  `domain` / `infrastructure` / `presentation`) repeats for every module.

### Negative / trade-offs
- **Boundaries are convention-enforced.** Go's package visibility helps, but discipline
  (and review/linting) is still required to stop someone importing another module's
  internals.
- **No DB-level referential integrity across modules.** Forbidding cross-module FKs
  means cross-module consistency is the application's responsibility, not the database's.
- **More indirection.** Contract clients and DTO mapping add boilerplate compared to
  calling a function or joining a table directly.
- **Shared fate at runtime.** A single process means one module can affect the whole
  app's resource use; horizontal scaling is whole-monolith, not per-module.

### Why a modular monolith over microservices
- The team is small and the deployment is intentionally one container with a host-level
  database (ADR-0001); microservices would add network hops, distributed transactions,
  and operational tooling we do not need at this scale.
- We still want the clean internal boundaries microservices are often adopted for — so
  we get them via the contracts-only rule and primitive-ID relations, while keeping
  in-process atomicity and a single, simple deployable.
- The contract-first module seams mean we can extract a service later **if** load or
  organizational scale ever demands it, without having paid the cost upfront.
