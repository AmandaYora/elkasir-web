import { lazy, Suspense } from "react";
import type { RouteObject } from "react-router-dom";
import { TenantAuthLayout } from "@/modules/auth/layouts/TenantAuthLayout";
import { PlatformAuthLayout } from "@/modules/platform/layouts/PlatformAuthLayout";
import { LoadingState } from "@/shared/components/feedback";
import { ROUTE_PATHS } from "./route-paths";

const LoginPage = lazy(() => import("@/modules/auth/pages/LoginPage"));
const PlatformLoginPage = lazy(() => import("@/modules/platform/pages/PlatformLoginPage"));
const PublicOrderPage = lazy(() => import("@/modules/self-order/pages/PublicOrderPage"));

// Public, no-auth routes. Login lives under TenantAuthLayout/PlatformAuthLayout (each redirects
// if already signed in, to their own dashboard); the customer self-order page has no layout (a
// standalone public surface).
export const publicRoutes: RouteObject[] = [
  {
    element: <TenantAuthLayout />,
    children: [{ path: ROUTE_PATHS.login, element: <LoginPage /> }],
  },
  {
    element: <PlatformAuthLayout />,
    children: [{ path: ROUTE_PATHS.platformLogin, element: <PlatformLoginPage /> }],
  },
  {
    path: ROUTE_PATHS.publicOrder,
    element: (
      <Suspense fallback={<LoadingState />}>
        <PublicOrderPage />
      </Suspense>
    ),
  },
];
