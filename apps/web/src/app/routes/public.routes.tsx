import { lazy, Suspense } from "react";
import type { RouteObject } from "react-router-dom";
import { AuthLayout } from "@/shared/layouts/AuthLayout";
import { LoadingState } from "@/shared/components/feedback";
import { ROUTE_PATHS } from "./route-paths";

const LoginPage = lazy(() => import("@/modules/auth/pages/LoginPage"));
const PublicOrderPage = lazy(() => import("@/modules/self-order/pages/PublicOrderPage"));

// Public, no-auth routes. Login lives under AuthLayout (redirects if already signed in);
// the customer self-order page has no layout (a standalone public surface).
export const publicRoutes: RouteObject[] = [
  {
    element: <AuthLayout />,
    children: [{ path: ROUTE_PATHS.login, element: <LoginPage /> }],
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
