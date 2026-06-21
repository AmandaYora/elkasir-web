import { Suspense } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { Store, Wallet, Receipt, QrCode, ArrowUpRight, type LucideIcon } from "lucide-react";
import { useAuthStore } from "@/shared/stores/auth.store";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { appBrand } from "@/shared/constants/brand";
import { LoadingState } from "@/shared/components/feedback";
import { cn } from "@/shared/lib/cn";

// Public auth shell (login). A split screen that speaks the dashboard's visual language:
// a brand panel previewing the product on the left, the routed form on the right.
// Redirects to the dashboard if already signed in.
export function AuthLayout() {
  const user = useAuthStore((s) => s.user);
  const status = useAuthStore((s) => s.status);

  if (status === "loading") {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <LoadingState label="Memuat…" />
      </div>
    );
  }
  if (user) return <Navigate to={ROUTE_PATHS.dashboard} replace />;

  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[1.05fr_1fr]">
      <BrandShowcase />

      {/* Form column */}
      <div className="flex min-h-screen flex-col bg-background">
        <div className="px-6 pt-6 lg:hidden">
          <BrandMark />
        </div>

        <div className="flex flex-1 items-center justify-center px-6 py-10">
          <div className="auth-rise w-full max-w-100">
            <Suspense fallback={<LoadingState />}>
              <Outlet />
            </Suspense>
          </div>
        </div>

        <footer className="px-6 pb-6 text-center lg:text-left">
          <p className="font-mono text-[11px] text-muted">
            © {new Date().getFullYear()} {appBrand} · Point of Sale F&amp;B
          </p>
        </footer>
      </div>
    </div>
  );
}

// The brand lockup, mirrored from the dashboard sidebar so the two surfaces read as one product.
function BrandMark({ variant = "dark" }: { variant?: "dark" | "light" }) {
  const light = variant === "light";
  return (
    <div className="flex items-center gap-2.5">
      <span
        className={cn(
          "flex h-9 w-9 items-center justify-center rounded-lg shadow-sm",
          light ? "bg-white/15 text-white backdrop-blur" : "bg-primary text-primary-foreground",
        )}
      >
        <Store className="h-4 w-4" />
      </span>
      <div className="flex flex-col">
        <span
          className={cn("text-sm font-semibold leading-tight", light ? "text-white" : "text-text")}
        >
          {appBrand}
        </span>
        <span className={cn("text-[11px] leading-tight", light ? "text-white/60" : "text-muted")}>
          Admin POS
        </span>
      </div>
    </div>
  );
}

// Left panel: a deep-blue ground that previews the dashboard the admin is signing into.
function BrandShowcase() {
  return (
    <div className="relative hidden overflow-hidden bg-primary lg:flex lg:flex-col lg:justify-between lg:p-12 xl:p-14">
      {/* Depth + a faint dot grid echoing the dashboard's quiet precision. */}
      <div
        aria-hidden
        className="absolute inset-0"
        style={{
          background:
            "radial-gradient(120% 120% at 0% 0%, rgba(255,255,255,0.14), transparent 46%), radial-gradient(100% 100% at 100% 100%, rgba(12,19,32,0.55), transparent 55%)",
        }}
      />
      <div
        aria-hidden
        className="absolute inset-0 opacity-60"
        style={{
          backgroundImage:
            "radial-gradient(circle at 1px 1px, rgba(255,255,255,0.16) 1px, transparent 0)",
          backgroundSize: "22px 22px",
        }}
      />

      <div className="relative">
        <BrandMark variant="light" />
      </div>

      <div className="relative max-w-md">
        <h2 className="font-display text-3xl font-extrabold leading-[1.1] tracking-tight text-white xl:text-4xl">
          Kelola seluruh tokomu dari satu dasbor.
        </h2>
        <p className="mt-4 text-sm leading-relaxed text-white/70">
          Pantau penjualan real-time, kelola produk dan meja, dan awasi arus kas — semua dari satu
          layar.
        </p>
        <DashboardPreview />
      </div>

      <div className="relative">
        <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-white/45">
          Dipercaya mengelola ratusan transaksi tiap hari
        </p>
      </div>
    </div>
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
