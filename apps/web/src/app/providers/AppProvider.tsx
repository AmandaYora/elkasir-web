import { useEffect, type ReactNode } from "react";
import { Toaster } from "sonner";
import { useAuthStore } from "@/shared/stores/auth.store";

// App-wide providers: restores the session on mount and mounts the toast portal.
export function AppProvider({ children }: { children: ReactNode }) {
  const restore = useAuthStore((s) => s.restore);

  useEffect(() => {
    void restore();
  }, [restore]);

  return (
    <>
      {children}
      <Toaster richColors position="top-right" />
    </>
  );
}
