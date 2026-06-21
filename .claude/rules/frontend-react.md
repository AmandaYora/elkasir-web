# Rule: Frontend React (applies to `apps/web/**`)

- Stack is locked: React 19, Tailwind 4, **react-router-dom** (lazy routes), **Zustand**, **Zod**,
  **Axios**, `@/*` alias. **Do NOT use TanStack** (Query/Router/Table) or add Redis/Memcache patterns.
- Feature code lives in `src/modules/<module>/` (`pages`, `components`, `services`, `schemas`, `stores`,
  `hooks`, `types`, `index.ts`). Generic, domain-agnostic UI lives in `src/shared/`.
- A component belongs to a module if it imports that module's types/services/stores/schemas/domain
  constants. Only move to `src/shared/components` when it is genuinely domain-agnostic. Shared UI must
  not reference domain statuses.
- **All HTTP** goes through `src/shared/services/http-client.ts` (one Axios instance). Modules must not
  create their own Axios instance. Module API functions live in `modules/<m>/services`. Paths use
  `/api/v1` via `import.meta.env.VITE_API_BASE_URL`.
- Replace TanStack Query with lightweight per-module hooks (`useState`/`useEffect` or small Zustand
  stores). Keep fetch functions in `services`.
- Routing: store paths in `app/routes/route-paths.ts`; separate public vs protected routes; lazy-load
  pages; wrap in Suspense; apply layouts at the route level.
- Zod schemas live in `modules/<m>/schemas`; derive form types with `z.infer` (don't duplicate types).
- Theme colors are centralized in `src/theme/` (CSS variables). Never hardcode brand colors across
  components; changing the theme must not require editing many files.
- Use the `frontend-design` skill as the UI authority when building/reshaping UI.
