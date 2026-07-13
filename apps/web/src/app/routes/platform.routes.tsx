import { lazy } from "react";
import type { RouteObject } from "react-router-dom";
import { PlatformLayout } from "@/modules/platform/layouts/PlatformLayout";
import { ROUTE_PATHS } from "./route-paths";

// Konsol Platform's protected pages — all lazy-loaded (code-split), same convention as
// protected.routes.tsx. Suspense lives in PlatformLayout's Outlet.
const PlatformOverviewPage = lazy(() => import("@/modules/platform/pages/PlatformOverviewPage"));
const PlatformTenantsPage = lazy(() => import("@/modules/platform/pages/PlatformTenantsPage"));
const PlatformTenantRevenuePage = lazy(
  () => import("@/modules/platform/pages/PlatformTenantRevenuePage"),
);
const PlatformWithdrawalsPage = lazy(
  () => import("@/modules/platform/pages/PlatformWithdrawalsPage"),
);
const PlatformWithdrawalHistoryPage = lazy(
  () => import("@/modules/platform/pages/PlatformWithdrawalHistoryPage"),
);
const PlatformPlansPage = lazy(() => import("@/modules/platform/pages/PlatformPlansPage"));
const PlatformUsersPage = lazy(() => import("@/modules/platform/pages/PlatformUsersPage"));
const PlatformPaymentConfigPage = lazy(
  () => import("@/modules/platform/pages/PlatformPaymentConfigPage"),
);
const PlatformPaymentClientsPage = lazy(
  () => import("@/modules/platform/pages/PlatformPaymentClientsPage"),
);

export const platformRoutes: RouteObject[] = [
  {
    // Unlike the tenant AppLayout wrapper (pathless — tenant dashboard IS site root "/"),
    // Konsol Platform is NOT the site root, so this wrapper needs its own `path` — otherwise
    // its pathless `index` route would resolve to "/" and collide with the tenant dashboard's.
    path: ROUTE_PATHS.platformDashboard,
    element: <PlatformLayout />,
    children: [
      { index: true, element: <PlatformOverviewPage /> },
      { path: ROUTE_PATHS.platformTenants, element: <PlatformTenantsPage /> },
      { path: ROUTE_PATHS.platformTenantsRevenue, element: <PlatformTenantRevenuePage /> },
      { path: ROUTE_PATHS.platformWithdrawals, element: <PlatformWithdrawalsPage /> },
      {
        path: ROUTE_PATHS.platformWithdrawalHistory,
        element: <PlatformWithdrawalHistoryPage />,
      },
      { path: ROUTE_PATHS.platformPlans, element: <PlatformPlansPage /> },
      { path: ROUTE_PATHS.platformUsers, element: <PlatformUsersPage /> },
      { path: ROUTE_PATHS.platformPaymentConfig, element: <PlatformPaymentConfigPage /> },
      { path: ROUTE_PATHS.platformPaymentClients, element: <PlatformPaymentClientsPage /> },
    ],
  },
];
