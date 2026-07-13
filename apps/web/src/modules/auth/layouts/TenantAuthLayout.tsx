import { Wallet, Receipt, QrCode, ArrowUpRight, type LucideIcon } from "lucide-react";
import { AuthLayout, BrandPanelShell } from "@/shared/layouts/AuthLayout";
import { useAuthStore } from "@/shared/stores/auth.store";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { cn } from "@/shared/lib/cn";

// Tenant (admin/staff) login shell — supplies the tenant-specific guard + brand copy/preview to
// the shared, domain-agnostic <AuthLayout/>. Renders pixel-identical to the pre-refactor version.
export function TenantAuthLayout() {
  const user = useAuthStore((s) => s.user);
  const status = useAuthStore((s) => s.status);

  return (
    <AuthLayout
      guard={{ isAuthenticated: !!user, status, redirectTo: ROUTE_PATHS.dashboard }}
      brandPanel={
        <BrandPanelShell
          subtitle="Admin POS"
          headline="Kelola seluruh tokomu dari satu dasbor."
          description="Pantau penjualan real-time, kelola produk dan meja, dan awasi arus kas — semua dari satu layar."
          preview={<DashboardPreview />}
          tagline="Dipercaya mengelola ratusan transaksi tiap hari"
        />
      }
    />
  );
}

// A glassy glimpse of the dashboard, built in the same StatCard idiom the app uses.
function DashboardPreview() {
  const bars = [45, 62, 50, 73, 58, 86, 70];
  const peak = bars.indexOf(Math.max(...bars));
  return (
    <div className="auth-rise mt-9 space-y-3" style={{ animationDelay: "120ms" }}>
      <div className="rounded-2xl border border-white/15 bg-white/10 p-5 backdrop-blur-md">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/15 text-white">
              <Wallet className="h-5 w-5" />
            </span>
            <div>
              <p className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/60">
                Pendapatan hari ini
              </p>
              <p className="mt-0.5 text-xl font-bold tabular-nums text-white">Rp 4.820.000</p>
            </div>
          </div>
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-white/15 px-2 py-1 text-[11px] font-semibold text-white">
            <ArrowUpRight className="h-3 w-3" /> 12,5%
          </span>
        </div>
        <div className="mt-4 flex h-12 items-end gap-1.5">
          {bars.map((h, i) => (
            <span
              key={i}
              className={cn("flex-1 rounded-t", i === peak ? "bg-white/55" : "bg-white/20")}
              style={{ height: `${h}%` }}
            />
          ))}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <MiniStat icon={Receipt} label="Transaksi" value="142" />
        <MiniStat icon={QrCode} label="QRIS" value="38%" />
      </div>
    </div>
  );
}

function MiniStat({
  icon: Icon,
  label,
  value,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-xl border border-white/15 bg-white/10 p-4 backdrop-blur-md">
      <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-white/15 text-white">
        <Icon className="h-4 w-4" />
      </span>
      <p className="mt-3 font-mono text-[10px] uppercase tracking-[0.16em] text-white/60">
        {label}
      </p>
      <p className="text-lg font-bold tabular-nums text-white">{value}</p>
    </div>
  );
}
