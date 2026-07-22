import { Suspense, useEffect, useState } from "react";
import { Navigate, Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  LayoutDashboard,
  Package,
  Tags,
  Receipt,
  Clock,
  Banknote,
  ArrowDownToLine,
  BarChart3,
  Users,
  LayoutGrid,
  ShieldCheck,
  Inbox,
  SlidersHorizontal,
  CreditCard,
  TriangleAlert,
} from "lucide-react";
import { useAuthStore } from "@/shared/stores/auth.store";
import { usePaymentLockStore } from "@/shared/stores/payment-lock.store";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { adminRoleLabel } from "@/shared/constants/brand";
import { AppSidebar, type NavGroup } from "./AppSidebar";
import { AppHeader } from "./AppHeader";
import { LoadingState } from "@/shared/components/feedback";

const tenantNavGroups: NavGroup[] = [
  {
    label: "Ikhtisar",
    items: [
      { title: "Dasbor", to: ROUTE_PATHS.dashboard, icon: LayoutDashboard, end: true },
      { title: "Produk", to: ROUTE_PATHS.products, icon: Package },
      { title: "Kategori Produk", to: ROUTE_PATHS.categories, icon: Tags },
      { title: "Transaksi", to: ROUTE_PATHS.transactions, icon: Receipt },
    ],
  },
  {
    label: "Operasional",
    items: [
      { title: "Pesanan Masuk", to: ROUTE_PATHS.incoming, icon: Inbox },
      { title: "Shift Staf", to: ROUTE_PATHS.shifts, icon: Clock },
      { title: "Meja", to: ROUTE_PATHS.tables, icon: LayoutGrid },
      { title: "Mutasi Kas", to: ROUTE_PATHS.cashMovements, icon: Banknote },
      // Penarikan disembunyikan sementara dari sidebar (frontend-only, lihat protected.routes.tsx).
      // { title: "Penarikan", to: ROUTE_PATHS.withdrawals, icon: ArrowDownToLine },
    ],
  },
  {
    label: "Analitik",
    items: [
      { title: "Statistik", to: ROUTE_PATHS.statistics, icon: BarChart3 },
      { title: "Staf", to: ROUTE_PATHS.staff, icon: Users },
      { title: "Pengguna", to: ROUTE_PATHS.users, icon: ShieldCheck },
    ],
  },
  {
    label: "Sistem",
    items: [
      { title: "Pengaturan", to: ROUTE_PATHS.settings, icon: SlidersHorizontal },
      { title: "Langganan", to: ROUTE_PATHS.subscription, icon: CreditCard },
    ],
  },
];

// Sidebar shown while the tenant's subscription package is inactive (§2.15) — every other
// tenant route redirects here, so only Langganan itself is worth linking to.
const lockedNavGroups: NavGroup[] = [
  {
    label: "Sistem",
    items: [{ title: "Langganan", to: ROUTE_PATHS.subscription, icon: CreditCard, end: true }],
  },
];

// Protected admin shell: guards the tenant session, then renders sidebar + header + routed
// page. Tenant-specific composition of the domain-agnostic AppSidebar/AppHeader — see
// PlatformLayout for Konsol Platform's twin.
export function AppLayout() {
  const user = useAuthStore((s) => s.user);
  const status = useAuthStore((s) => s.status);
  const logout = useAuthStore((s) => s.logout);
  const locked = usePaymentLockStore((s) => s.locked);
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  // Close the off-canvas sidebar whenever the route changes (link click already closes it via
  // AppSidebar's onClick, but this also covers back/forward navigation and redirects).
  useEffect(() => {
    setMobileNavOpen(false);
  }, [location.pathname]);

  if (status === "loading") {
    return (
      <div className="flex h-screen items-center justify-center">
        <LoadingState label="Memulihkan sesi…" />
      </div>
    );
  }
  if (!user) return <Navigate to={ROUTE_PATHS.login} replace />;

  const onLogout = async () => {
    await logout();
    navigate(ROUTE_PATHS.login, { replace: true });
  };

  // §2.15: a package-inactive admin can always reach Langganan, nothing else. Redirect away
  // from any other route instead of showing a raw 402 error on whatever page they were on.
  if (locked && location.pathname !== ROUTE_PATHS.subscription) {
    return <Navigate to={ROUTE_PATHS.subscription} replace />;
  }

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <AppSidebar
        groups={locked ? lockedNavGroups : tenantNavGroups}
        subtitle="Admin POS"
        mobileOpen={mobileNavOpen}
        onClose={() => setMobileNavOpen(false)}
      />
      <div className="flex min-w-0 flex-1 flex-col">
        <AppHeader
          user={{ name: user.name, roleLabel: adminRoleLabel[user.role] ?? user.role }}
          onLogout={onLogout}
          onMenuClick={() => setMobileNavOpen(true)}
        />
        {locked && (
          <div className="flex items-center gap-2 border-b border-warning/30 bg-warning-soft px-4 py-2 text-sm text-warning md:px-6">
            <TriangleAlert className="h-4 w-4 shrink-0" />
            Paket langganan toko Anda tidak aktif. Perbarui paket langganan untuk melanjutkan.
          </div>
        )}
        <main className="flex-1 overflow-y-auto">
          <Suspense fallback={<LoadingState />}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
