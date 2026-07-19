import {
  LayoutDashboard,
  Building2,
  TrendingUp,
  ArrowDownToLine,
  History,
  Layers,
  Users,
  Settings2,
} from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { NavGroup } from "@/shared/layouts/AppSidebar";

// Konsol Platform's sidebar grouping (PLAN.md §2.12) — mirrors the tenant sidebar's own grouped
// convention, rendered by the same domain-agnostic <AppSidebar/>.
export const platformNavGroups: NavGroup[] = [
  {
    label: "Ikhtisar",
    items: [
      { title: "Ringkasan", to: ROUTE_PATHS.platformDashboard, icon: LayoutDashboard, end: true },
    ],
  },
  {
    label: "Tenant",
    items: [
      { title: "Tenant", to: ROUTE_PATHS.platformTenants, icon: Building2, end: true },
      { title: "Revenue Tenant", to: ROUTE_PATHS.platformTenantsRevenue, icon: TrendingUp },
    ],
  },
  {
    label: "Keuangan",
    items: [
      { title: "Penarikan", to: ROUTE_PATHS.platformWithdrawals, icon: ArrowDownToLine, end: true },
      { title: "Riwayat Penarikan", to: ROUTE_PATHS.platformWithdrawalHistory, icon: History },
    ],
  },
  {
    label: "Sistem",
    items: [
      { title: "Paket", to: ROUTE_PATHS.platformPlans, icon: Layers },
      { title: "User Platform", to: ROUTE_PATHS.platformUsers, icon: Users },
      { title: "Konfigurasi Pembayaran", to: ROUTE_PATHS.platformPaymentConfig, icon: Settings2 },
    ],
  },
];
