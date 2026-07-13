import { useEffect, type ReactNode } from "react";
import { Toaster } from "sonner";
import { useAuthStore } from "@/shared/stores/auth.store";
import { usePlatformAuthStore } from "@/modules/platform/stores/platform-auth.store";
import { usePaymentLockStore } from "@/shared/stores/payment-lock.store";
import { setOnPaymentRequired } from "@/shared/services/http-client";

// App-wide providers: restores BOTH sessions on mount (tenant + platform are fully separate
// identity domains, §2.1 — either, both, or neither may be signed in at once), wires the
// subscription-lock callback (§2.15), and mounts the toast portal.
export function AppProvider({ children }: { children: ReactNode }) {
  const restore = useAuthStore((s) => s.restore);
  const restorePlatform = usePlatformAuthStore((s) => s.restore);

  useEffect(() => {
    void restore();
    void restorePlatform();
    setOnPaymentRequired(() => usePaymentLockStore.getState().setLocked(true));
  }, [restore, restorePlatform]);

  return (
    <>
      {children}
      <Toaster richColors position="top-right" />
    </>
  );
}
