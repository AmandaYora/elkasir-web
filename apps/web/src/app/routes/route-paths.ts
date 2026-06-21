// Central route path registry. Reference these instead of hardcoding URL strings.
export const ROUTE_PATHS = {
  // public
  login: "/login",
  publicOrder: "/order/:code",
  publicOrderTo: (code: string) => `/order/${encodeURIComponent(code)}`,

  // protected (admin shell)
  dashboard: "/",
  products: "/products",
  categories: "/categories",
  transactions: "/transactions",
  incoming: "/incoming",
  shifts: "/shifts",
  tables: "/tables",
  cashMovements: "/cash-movements",
  withdrawals: "/withdrawals",
  statistics: "/statistics",
  staff: "/cashiers",
  users: "/users",
} as const;
