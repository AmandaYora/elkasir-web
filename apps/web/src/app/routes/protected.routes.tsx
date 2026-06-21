import { lazy } from "react";
import type { RouteObject } from "react-router-dom";
import { AppLayout } from "@/shared/layouts/AppLayout";
import { ROUTE_PATHS } from "./route-paths";

// All admin pages are lazy-loaded (code-split). Suspense lives in AppLayout's Outlet.
const DashboardPage = lazy(() => import("@/modules/dashboard/pages/DashboardPage"));
const ProductsPage = lazy(() => import("@/modules/products/pages/ProductsPage"));
const CategoriesPage = lazy(() => import("@/modules/categories/pages/CategoriesPage"));
const TransactionsPage = lazy(() => import("@/modules/transactions/pages/TransactionsPage"));
const IncomingOrdersPage = lazy(() => import("@/modules/self-order/pages/IncomingOrdersPage"));
const ShiftsPage = lazy(() => import("@/modules/shifts/pages/ShiftsPage"));
const TablesPage = lazy(() => import("@/modules/tables/pages/TablesPage"));
const CashMovementsPage = lazy(() => import("@/modules/cash-movements/pages/CashMovementsPage"));
const WithdrawalsPage = lazy(() => import("@/modules/withdrawals/pages/WithdrawalsPage"));
const StatisticsPage = lazy(() => import("@/modules/statistics/pages/StatisticsPage"));
const StaffPage = lazy(() => import("@/modules/staff/pages/StaffPage"));
const UsersPage = lazy(() => import("@/modules/users/pages/UsersPage"));

export const protectedRoutes: RouteObject[] = [
  {
    element: <AppLayout />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: ROUTE_PATHS.products, element: <ProductsPage /> },
      { path: ROUTE_PATHS.categories, element: <CategoriesPage /> },
      { path: ROUTE_PATHS.transactions, element: <TransactionsPage /> },
      { path: ROUTE_PATHS.incoming, element: <IncomingOrdersPage /> },
      { path: ROUTE_PATHS.shifts, element: <ShiftsPage /> },
      { path: ROUTE_PATHS.tables, element: <TablesPage /> },
      { path: ROUTE_PATHS.cashMovements, element: <CashMovementsPage /> },
      { path: ROUTE_PATHS.withdrawals, element: <WithdrawalsPage /> },
      { path: ROUTE_PATHS.statistics, element: <StatisticsPage /> },
      { path: ROUTE_PATHS.staff, element: <StaffPage /> },
      { path: ROUTE_PATHS.users, element: <UsersPage /> },
    ],
  },
];
