import { Suspense, type ReactNode } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { appBrand } from "@/shared/constants/brand";
import { LoadingState } from "@/shared/components/feedback";
import { cn } from "@/shared/lib/cn";

export interface AuthGuard {
  isAuthenticated: boolean;
  status: "loading" | "ready";
  redirectTo: string;
}

export interface AuthLayoutProps {
  guard: AuthGuard;
  /** Desktop brand panel (left column) — build with <BrandPanelShell/>. */
  brandPanel: ReactNode;
  /** Mobile-only top bar subtitle (BrandMark) — default preserves the tenant's existing copy. */
  subtitle?: string;
  footerTagline?: string;
}

// Public auth shell (login) — domain-agnostic. A split screen: a brand panel on the left
// (supplied by the caller — tenant vs platform each build their own via BrandPanelShell), the
// routed form on the right. Redirects to `guard.redirectTo` if already signed in. Per PLAN.md
// §2.2: login pages differ (copy/preview via the props here); everything after login reuses
// the same components.
export function AuthLayout({
  guard,
  brandPanel,
  subtitle = "Admin POS",
  footerTagline = "Point of Sale F&B",
}: AuthLayoutProps) {
  if (guard.status === "loading") {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <LoadingState label="Memuat…" />
      </div>
    );
  }
  if (guard.isAuthenticated) return <Navigate to={guard.redirectTo} replace />;

  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[1.05fr_1fr]">
      {brandPanel}

      {/* Form column */}
      <div className="flex min-h-screen flex-col bg-background">
        <div className="px-6 pt-6 lg:hidden">
          <BrandMark subtitle={subtitle} />
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
            © {new Date().getFullYear()} {appBrand} · {footerTagline}
          </p>
        </footer>
      </div>
    </div>
  );
}

// The brand lockup, mirrored from the dashboard sidebar so the two surfaces read as one product.
export function BrandMark({
  variant = "dark",
  subtitle = "Admin POS",
}: {
  variant?: "dark" | "light";
  subtitle?: string;
}) {
  const light = variant === "light";
  return (
    <div className="flex items-center gap-2.5">
      <img src="/elkasir-logo.png" alt={appBrand} className="h-9 w-9 shrink-0" />
      <div className="flex flex-col">
        <span
          className={cn("text-sm font-semibold leading-tight", light ? "text-white" : "text-text")}
        >
          {appBrand}
        </span>
        <span className={cn("text-[11px] leading-tight", light ? "text-white/60" : "text-muted")}>
          {subtitle}
        </span>
      </div>
    </div>
  );
}

export interface BrandPanelShellProps {
  /** BrandMark subtitle inside the panel — default preserves the tenant's existing copy. */
  subtitle?: string;
  headline: string;
  description: string;
  preview?: ReactNode;
  tagline: string;
}

// Left panel: a deep-blue ground previewing whatever the caller is signing into (tenant
// dashboard vs Konsol Platform) — headline/description/preview/tagline are the ONLY things
// that differ between login pages (§2.2); the shell itself (gradients, dot grid, layout) is shared.
export function BrandPanelShell({
  subtitle = "Admin POS",
  headline,
  description,
  preview,
  tagline,
}: BrandPanelShellProps) {
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
        <BrandMark variant="light" subtitle={subtitle} />
      </div>

      <div className="relative max-w-md">
        <h2 className="font-display text-3xl font-extrabold leading-[1.1] tracking-tight text-white xl:text-4xl">
          {headline}
        </h2>
        <p className="mt-4 text-sm leading-relaxed text-white/70">{description}</p>
        {preview}
      </div>

      <div className="relative">
        <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-white/45">{tagline}</p>
      </div>
    </div>
  );
}
