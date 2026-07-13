import { ShieldCheck, Building2, Wallet, type LucideIcon } from "lucide-react";
import { AuthLayout, BrandPanelShell } from "@/shared/layouts/AuthLayout";
import { usePlatformAuthStore } from "@/modules/platform/stores/platform-auth.store";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

// Platform (superadmin) login shell — twin of TenantAuthLayout, but guards on
// usePlatformAuthStore and supplies Konsol Platform's own copy/preview (§2.2).
export function PlatformAuthLayout() {
  const user = usePlatformAuthStore((s) => s.user);
  const status = usePlatformAuthStore((s) => s.status);

  return (
    <AuthLayout
      guard={{ isAuthenticated: !!user, status, redirectTo: ROUTE_PATHS.platformDashboard }}
      subtitle="Konsol Platform"
      footerTagline="Konsol Platform"
      brandPanel={
        <BrandPanelShell
          subtitle="Konsol Platform"
          headline="Kelola seluruh tenant dari satu konsol."
          description="Pantau langganan, saldo tenant yang belum dicairkan, dan proses pencairan dana dengan jejak audit yang jelas."
          preview={<ReconciliationPreview />}
          tagline="Satu konsol untuk seluruh tenant Elkasir"
        />
      }
    />
  );
}

// A glassy glimpse of the reconciliation dashboard — mirrors TenantAuthLayout's DashboardPreview
// idiom, but with platform-relevant figures (subscription + tenant balance) instead of sales.
function ReconciliationPreview() {
  return (
    <div className="auth-rise mt-9 space-y-3" style={{ animationDelay: "120ms" }}>
      <div className="rounded-2xl border border-white/15 bg-white/10 p-5 backdrop-blur-md">
        <div className="flex items-center gap-3">
          <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/15 text-white">
            <ShieldCheck className="h-5 w-5" />
          </span>
          <div>
            <p className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/60">
              Total termonitor
            </p>
            <p className="mt-0.5 text-xl font-bold tabular-nums text-white">Rp 18.400.000</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <MiniStat icon={Building2} label="Tenant aktif" value="24" />
        <MiniStat icon={Wallet} label="Saldo belum cair" value="Rp 6,2jt" />
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
