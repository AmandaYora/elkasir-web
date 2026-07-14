// Central route path registry. Reference these instead of hardcoding URL strings.
export const ROUTE_PATHS = {
  // public
  login: "/login",
  platformLogin: "/platform/login",
  // Marketing homepage — static, frontend-only, no backend integration.
  homepage: "/homepage",
  homepageTerms: "/homepage/syarat-ketentuan",
  homepageContact: "/homepage/kontak",
  // {slug} toko wajib: kode meja cuma unik per-toko (lihat migration 000016), jadi tenant
  // harus dari slug, bukan cuma kode meja.
  publicOrder: "/order/:slug/:code",
  publicOrderTo: (slug: string, code: string) =>
    `/order/${encodeURIComponent(slug)}/${encodeURIComponent(code)}`,

  // protected (admin shell)
  dashboard: "/",
  subscription: "/subscription",

  // protected (Konsol Platform shell — see platform.routes.tsx, Phase F3)
  platformDashboard: "/platform",
  platformTenants: "/platform/tenants",
  platformTenantsRevenue: "/platform/tenants/revenue",
  platformWithdrawals: "/platform/withdrawals",
  platformWithdrawalHistory: "/platform/withdrawals/history",
  platformPlans: "/platform/plans",
  platformUsers: "/platform/users",
  platformPaymentConfig: "/platform/payment-config",
  platformPaymentClients: "/platform/payment-clients",
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
  settings: "/settings",
} as const;
