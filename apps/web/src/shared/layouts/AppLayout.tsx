import { Suspense } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { useAuthStore } from "@/shared/stores/auth.store";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { AppSidebar } from "./AppSidebar";
import { AppHeader } from "./AppHeader";
import { LoadingState } from "@/shared/components/feedback";

// Protected admin shell: guards the session, then renders sidebar + header + routed page.
export function AppLayout() {
  const user = useAuthStore((s) => s.user);
  const status = useAuthStore((s) => s.status);

  if (status === "loading") {
    return (
      <div className="flex h-screen items-center justify-center">
        <LoadingState label="Memulihkan sesi…" />
      </div>
    );
  }
  if (!user) return <Navigate to={ROUTE_PATHS.login} replace />;

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <AppSidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <AppHeader />
        <main className="flex-1 overflow-y-auto">
          <Suspense fallback={<LoadingState />}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
