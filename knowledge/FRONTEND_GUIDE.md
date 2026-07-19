# Frontend Guide (`apps/web`)

React 19 + Tailwind 4 SPA. **No TanStack** (router or query), **no Turborepo**. Build is plain Vite.

## Stack

- **react-router-dom** with **lazy-loaded** route components.
- **Zustand** for state (global in `shared/stores`, module state in `modules/<m>/stores`).
- **Zod** for schemas/validation (`modules/<m>/schemas`); derive form types with `z.infer`.
- **Axios** as the single HTTP client (`shared/services/http-client.ts`).
- `@/*` import alias → `src/*`. Centralized theme in `src/theme/`.

## Structure

```txt
apps/web/src/
├── app/
│   ├── App.tsx
│   ├── providers/            # AppProvider, RouterProvider
│   └── routes/               # index.tsx, public.routes.tsx, protected.routes.tsx, route-paths.ts
├── modules/<module>/         # auth, dashboard, products, categories, tables, staff, users,
│   ├── pages/                #   transactions, shifts, cash-movements, withdrawals, incoming,
│   ├── components/           #   statistics, self-order, subscription, platform (Konsol Platform)
│   ├── services/             # module API calls (use the shared http-client)
│   ├── schemas/              # Zod schemas
│   ├── stores/               # Zustand stores for module state
│   ├── hooks/                # data hooks (useProducts, …) replacing TanStack Query
│   ├── types/
│   └── index.ts
├── shared/
│   ├── components/ui/        # generic, domain-agnostic UI kit (button, input, modal, table, …)
│   ├── components/feedback/  # LoadingState, EmptyState, ErrorState
│   ├── layouts/              # AppLayout (admin shell), AuthLayout, etc.
│   ├── services/             # http-client.ts, api-endpoints.ts
│   ├── stores/               # global stores (auth/session)
│   ├── hooks/                # generic hooks
│   ├── lib/                  # cn.ts, formatter.ts, storage.ts
│   ├── types/                # api.ts, pagination.ts, common.ts
│   └── constants/
├── theme/                    # colors.ts, theme.css, index.ts
├── styles/globals.css
└── main.tsx
```

## Rules

- **HTTP**: all requests go through `shared/services/http-client.ts` (one Axios instance, base URL
  `import.meta.env.VITE_API_BASE_URL`). Modules never create their own Axios instance. Module
  calls live in `modules/<m>/services`. Paths are under `/api/v1`.
- **Two identity domains, one Axios instance**: the tenant admin dashboard and Konsol Platform
  (superadmin) are **fully separate sessions** — separate token storage keys, separate Zustand
  stores (`shared/stores/auth.store.ts` vs `modules/platform/stores/platform-auth.store.ts`),
  separate 401-refresh handling. Every request explicitly declares `tokenDomain: "tenant" |
  "platform"` (default `"tenant"`); logging into one must never authenticate the other. A `402`
  response (tenant domain only — package-inactive) is handled separately from `401`: it redirects
  to Langganan instead of logging the user out (`shared/stores/payment-lock.store.ts`).
- **Data fetching**: TanStack Query is removed. Use lightweight hooks (`useState` + `useEffect`, or a
  small `useAsync`/Zustand store) per module. Keep request functions in `services`.
- **Components**: a component stays in its module if it touches that module's types/services/stores/
  schemas/domain constants. Move to `shared/components` only when truly domain-agnostic. Shared UI must
  not know domain statuses (`ORDER_PAID`, `STOCK_LOW`, …).
- **Routing**: paths in `app/routes/route-paths.ts`; split public vs protected routes; lazy-load pages;
  wrap them in Suspense with a loading fallback; layouts applied at the route level.
- **Theme**: colors centralized in `src/theme/` (CSS variables in `theme.css`, tokens in `colors.ts`).
  `styles/globals.css` imports Tailwind + theme. Don't hardcode brand colors across components.
- **Auth/session**: a Zustand `authStore` (tenant) and a twin `usePlatformAuthStore` (Konsol
  Platform) each hold their own session user + token lifecycle (login, restore via `/auth/me`,
  logout) — see "Two identity domains" above. Protected routes read the matching store; the
  http-client reads the matching token for the Authorization header.
- **Shared layout components take props, not implicit store reads**: `AppSidebar`/`AppHeader`/
  `AppLayout` accept `groups`/`user`/`onLogout`/a session-guard config as props rather than
  importing `useAuthStore` internally — this is what lets `PlatformLayout` reuse the same shell
  components against `usePlatformAuthStore` instead of duplicating the layout.

## Self-order (public) pages

Customer-facing self-order pages are **public** (no auth): table menu, place order, order status. They
use the same http-client with `auth: false`-style requests to `/api/v1/public/...`.
