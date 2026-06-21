import { NavLink } from "react-router-dom";
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
  Store,
  LayoutGrid,
  ShieldCheck,
  Inbox,
} from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { appBrand } from "@/shared/constants/brand";
import { cn } from "@/shared/lib/cn";

type NavItem = { title: string; to: string; icon: React.ComponentType<{ className?: string }> };

const groups: { label: string; items: NavItem[] }[] = [
  {
    label: "Ikhtisar",
    items: [
      { title: "Dasbor", to: ROUTE_PATHS.dashboard, icon: LayoutDashboard },
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
      { title: "Penarikan", to: ROUTE_PATHS.withdrawals, icon: ArrowDownToLine },
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
];

export function AppSidebar() {
  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-border bg-surface">
      <div className="flex items-center gap-2.5 border-b border-border px-4 py-4">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm">
          <Store className="h-4 w-4" />
        </div>
        <div className="flex min-w-0 flex-col">
          <span className="truncate text-sm font-semibold leading-tight text-text">{appBrand}</span>
          <span className="truncate text-[11px] text-muted">Admin POS</span>
        </div>
      </div>

      <nav className="flex-1 overflow-y-auto py-2">
        {groups.map((group) => (
          <div key={group.label} className="px-2 py-1.5">
            <p className="px-3 pb-1 text-[11px] font-medium uppercase tracking-wider text-muted">{group.label}</p>
            <ul className="space-y-0.5">
              {group.items.map((item) => (
                <li key={item.title}>
                  <NavLink
                    to={item.to}
                    end={item.to === ROUTE_PATHS.dashboard}
                    className={({ isActive }: { isActive: boolean }) =>
                      cn(
                        "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors",
                        isActive
                          ? "bg-primary-soft font-medium text-primary"
                          : "text-text hover:bg-surface-muted",
                      )
                    }
                  >
                    <item.icon className="h-4 w-4 shrink-0" />
                    <span>{item.title}</span>
                  </NavLink>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </nav>
    </aside>
  );
}
