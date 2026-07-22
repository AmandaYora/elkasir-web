import { Suspense, useEffect, useState } from "react";
import { Navigate, Outlet, useLocation, useNavigate } from "react-router-dom";
import { usePlatformAuthStore } from "@/modules/platform/stores/platform-auth.store";
import { platformNavGroups } from "@/modules/platform/config/platformNav";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { AppSidebar } from "@/shared/layouts/AppSidebar";
import { AppHeader } from "@/shared/layouts/AppHeader";
import { LoadingState } from "@/shared/components/feedback";

// Konsol Platform's protected shell — twin of AppLayout, guards on usePlatformAuthStore,
// supplies platformNavGroups + the "Konsol Platform" subtitle to the same domain-agnostic
// AppSidebar/AppHeader (§2.2/§2.3 — same visual identity as the tenant dashboard, only the
// subtitle differs).
export function PlatformLayout() {
  const user = usePlatformAuthStore((s) => s.user);
  const status = usePlatformAuthStore((s) => s.status);
  const logout = usePlatformAuthStore((s) => s.logout);
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

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
  if (!user) return <Navigate to={ROUTE_PATHS.platformLogin} replace />;

  const onLogout = async () => {
    await logout();
    navigate(ROUTE_PATHS.platformLogin, { replace: true });
  };

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <AppSidebar
        groups={platformNavGroups}
        subtitle="Konsol Platform"
        mobileOpen={mobileNavOpen}
        onClose={() => setMobileNavOpen(false)}
      />
      <div className="flex min-w-0 flex-1 flex-col">
        <AppHeader
          user={{ name: user.name, roleLabel: "Superadmin" }}
          onLogout={onLogout}
          onMenuClick={() => setMobileNavOpen(true)}
        />
        <main className="flex-1 overflow-y-auto">
          <Suspense fallback={<LoadingState />}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
