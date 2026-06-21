# Knowledge Base — Index

This is the routing table for Elkasir's project knowledge. The skill holds the *standard*;
these files hold *this project*. Read the file matching your task before changing code.

| File | When to read |
|---|---|
| [PROJECT_BRIEF.md](PROJECT_BRIEF.md) | Onboarding — what Elkasir is, who uses it, why it exists. |
| [PRODUCT_REQUIREMENTS.md](PRODUCT_REQUIREMENTS.md) | Feature scope and user stories. |
| [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md) | Apps, modules, boundaries, data flow, deployment. |
| [MODULE_MAP.md](MODULE_MAP.md) | A module's responsibility and the public contract it exposes. |
| [DOMAIN_GLOSSARY.md](DOMAIN_GLOSSARY.md) | Definitions of domain terms used across code and docs. |
| [API_GUIDE.md](API_GUIDE.md) | API conventions: `/api/v1`, response envelopes, auth, pagination. |
| [DATABASE_GUIDE.md](DATABASE_GUIDE.md) | Table ownership per module, primitive-ID relations, migrations + sqlc. |
| [BACKEND_GUIDE.md](BACKEND_GUIDE.md) | Go backend conventions, module layout, Unit-of-Work, error handling. |
| [FRONTEND_GUIDE.md](FRONTEND_GUIDE.md) | React conventions: routing, stores, services, theme, UI kit. |
| [decisions/](decisions/) | Architecture Decision Records (ADRs). |

## Quick orientation

- **Monorepo**: `apps/web` (React 19) + `apps/api` (Go modular monolith) + `packages/api-contract` + `packages/shared`. Build with plain npm workspaces — **no Turborepo**.
- **API** at `/api/v1`; one container serves SPA + API; MySQL is host-level.
- **Boundaries**: only a module's `contracts/` is public; relations are primitive IDs; no cross-module joins/FKs. See [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md) and [MODULE_MAP.md](MODULE_MAP.md).
