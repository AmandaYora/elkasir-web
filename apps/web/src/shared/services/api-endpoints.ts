// Central registry of API paths (relative to VITE_API_BASE_URL, e.g. /api/v1).
// Modules reference these instead of hardcoding strings.
export const endpoints = {
  auth: {
    adminLogin: "/auth/admin/login",
    staffLogin: "/auth/staff/login",
    refresh: "/auth/refresh",
    logout: "/auth/logout",
    me: "/auth/me",
  },
  products: "/products",
  categories: "/categories",
  tables: "/tables",
  staff: "/staff",
  adminUsers: "/admin-users",
  transactions: "/transactions",
  shifts: "/shifts",
  cashMovements: "/cash-movements",
  withdrawals: "/withdrawals",
  reports: {
    dashboard: "/reports/dashboard",
    sales: "/reports/sales",
    topProducts: "/reports/top-products",
    salesByCategory: "/reports/sales-by-category",
    paymentDistribution: "/reports/payment-distribution",
    staffPerformance: "/reports/staff-performance",
  },
  selfOrders: "/self-orders",
  publicOrder: "/public/order",
  uploads: "/uploads",
  settings: "/settings",
} as const;
