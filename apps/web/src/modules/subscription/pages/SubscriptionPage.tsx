import { useState } from "react";
import { CalendarClock, Layers } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/shared/components/ui/button";
import { Badge } from "@/shared/components/ui/badge";
import { Card, CardContent } from "@/shared/components/ui/card";
import { LoadingState, ErrorState } from "@/shared/components/feedback";
import { useAsync } from "@/shared/hooks/useAsync";
import { useAuthStore } from "@/shared/stores/auth.store";
import { usePaymentLockStore } from "@/shared/stores/payment-lock.store";
import { subscriptionService } from "@/modules/subscription/services/subscription.service";
import { SubscriptionQrisPanel } from "@/modules/subscription/components/SubscriptionQrisPanel";
import type { CheckoutResult, Plan } from "@/modules/subscription/types/subscription.types";

const formatRupiah = (n: number) => `Rp ${n.toLocaleString("id-ID")}`;

const statusLabel: Record<string, string> = {
  none: "Belum berlangganan",
  trial: "Masa percobaan",
  active: "Aktif",
  past_due: "Jatuh tempo",
  expired: "Berakhir",
  canceled: "Dibatalkan",
};

function daysRemaining(periodEnd?: string): number | null {
  if (!periodEnd) return null;
  return Math.ceil((new Date(periodEnd).getTime() - Date.now()) / 86_400_000);
}

// Langganan — deliberately minimal (§2.4): current plan + countdown + upgrade CTA (only if a
// pricier plan exists) + manual "Cek status pembayaran" (no polling, no SSE). `status==="none"`
// shows a plan-picker. Checkout/upgrade is owner-only in the UI (backend re-checks regardless).
export default function SubscriptionPage() {
  const user = useAuthStore((s) => s.user);
  const isOwner = user?.role === "owner";

  const subQuery = useAsync(() => subscriptionService.getCurrent(), []);
  const plansQuery = useAsync(() => subscriptionService.listPlans(), []);
  const [checkoutResult, setCheckoutResult] = useState<CheckoutResult | null>(null);
  const [checkingOutId, setCheckingOutId] = useState<string | null>(null);
  const [checkingStatus, setCheckingStatus] = useState(false);

  const sub = subQuery.data;
  const plans = plansQuery.data ?? [];
  const currentPlan = plans.find((p) => p.id === sub?.planId);

  const doCheckout = async (planId: string) => {
    setCheckingOutId(planId);
    try {
      const result = await subscriptionService.checkout(planId);
      setCheckoutResult(result);
      toast.success("Tagihan berhasil dibuat. Silakan bayar via QRIS.");
    } catch {
      toast.error("Gagal membuat tagihan langganan. Coba lagi.");
    } finally {
      setCheckingOutId(null);
    }
  };

  const checkStatus = async () => {
    setCheckingStatus(true);
    try {
      const fresh = await subscriptionService.getCurrent();
      subQuery.setData(fresh);
      if (fresh.status === "active") {
        setCheckoutResult(null);
        usePaymentLockStore.getState().setLocked(false);
        toast.success("Pembayaran terkonfirmasi — langganan aktif");
      } else {
        toast.info("Belum ada pembayaran baru terdeteksi.");
      }
    } catch {
      toast.error("Gagal memeriksa status. Coba lagi.");
    } finally {
      setCheckingStatus(false);
    }
  };

  if (subQuery.loading || plansQuery.loading) {
    return (
      <div className="p-4 md:p-6">
        <LoadingState />
      </div>
    );
  }
  if (subQuery.error || plansQuery.error || !sub) {
    return (
      <div className="p-4 md:p-6">
        <ErrorState
          message="Gagal memuat data langganan. Coba lagi."
          onRetry={() => {
            subQuery.refetch();
            plansQuery.refetch();
          }}
        />
      </div>
    );
  }

  // A checkout is in flight and still pending — show the QR, nothing else.
  if (checkoutResult && checkoutResult.invoice.status === "pending") {
    return (
      <div className="mx-auto max-w-md space-y-4 p-4 md:p-6">
        <div>
          <h2 className="text-lg font-semibold text-text">Langganan</h2>
          <p className="text-sm text-muted">Selesaikan pembayaran untuk mengaktifkan paket.</p>
        </div>
        <SubscriptionQrisPanel
          amount={checkoutResult.invoice.amount}
          qrString={checkoutResult.qrString}
          qrImageUrl={checkoutResult.qrImageUrl}
          checking={checkingStatus}
          onCheckStatus={checkStatus}
        />
      </div>
    );
  }

  // A renewal-only plan (e.g. "Premium Contributor") explicitly hides every other plan — not
  // just "pricier" upgrade options, ALL of them — since switching away from it is blocked
  // server-side regardless (subscription/application.Service.validatePlanSwitch). Checked via
  // sub.planRenewalOnly directly, not inferred from currentPlan being absent from the
  // active-only `plans` list — that would also happen to be true today (the plan is hidden) but
  // isn't the actual rule, and wouldn't hold for a hypothetical active-yet-renewal-only plan.
  const upgradeOptions =
    sub.planRenewalOnly || !currentPlan
      ? []
      : plans.filter((p) => p.isActive && p.id !== currentPlan.id && p.price > currentPlan.price);

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4 md:p-6">
      <div>
        <h2 className="text-lg font-semibold text-text">Langganan</h2>
        <p className="text-sm text-muted">Status paket langganan toko Anda ke Elkasir.</p>
      </div>

      {sub.status === "none" ? (
        <PlanPicker
          plans={plans}
          isOwner={isOwner}
          checkingOutId={checkingOutId}
          onPick={doCheckout}
        />
      ) : (
        <>
          <Card>
            <CardContent className="space-y-4 p-5">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-primary-soft text-primary">
                    <Layers className="h-5 w-5" />
                  </div>
                  <div>
                    <p className="text-sm font-semibold text-text">
                      {sub.planName ?? currentPlan?.name ?? "Paket tidak diketahui"}
                    </p>
                    <p className="text-xs text-muted">
                      {sub.planPrice != null ? formatRupiah(sub.planPrice) : ""}
                    </p>
                  </div>
                </div>
                <Badge tone={sub.status === "active" ? "success" : "warning"}>
                  {statusLabel[sub.status] ?? sub.status}
                </Badge>
              </div>

              {sub.currentPeriodEnd && (
                <div className="flex items-center gap-2 rounded-md bg-surface-muted px-3 py-2 text-sm text-muted">
                  <CalendarClock className="h-4 w-4 shrink-0" />
                  {(() => {
                    const days = daysRemaining(sub.currentPeriodEnd);
                    if (days === null) return null;
                    return days >= 0
                      ? `${days} hari lagi hingga ${new Date(sub.currentPeriodEnd!).toLocaleDateString("id-ID")}`
                      : `Sudah berakhir sejak ${new Date(sub.currentPeriodEnd!).toLocaleDateString("id-ID")}`;
                  })()}
                </div>
              )}

              {isOwner ? (
                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" onClick={checkStatus} loading={checkingStatus}>
                    Cek Status Pembayaran
                  </Button>
                  {sub.planId && (
                    <Button
                      loading={checkingOutId === sub.planId}
                      disabled={checkingOutId !== null && checkingOutId !== sub.planId}
                      onClick={() => doCheckout(sub.planId!)}
                    >
                      Perpanjang
                    </Button>
                  )}
                </div>
              ) : (
                <p className="text-xs text-muted">
                  Hubungi pemilik toko untuk mengelola langganan.
                </p>
              )}
            </CardContent>
          </Card>

          {upgradeOptions.length > 0 && (
            <div className="space-y-2">
              <h3 className="text-sm font-semibold text-text">Upgrade Paket</h3>
              <PlanPicker
                plans={upgradeOptions}
                isOwner={isOwner}
                checkingOutId={checkingOutId}
                onPick={doCheckout}
                ctaLabel="Upgrade"
              />
            </div>
          )}
        </>
      )}
    </div>
  );
}

function PlanPicker({
  plans,
  isOwner,
  checkingOutId,
  onPick,
  ctaLabel = "Pilih Paket",
}: {
  plans: Plan[];
  isOwner: boolean;
  checkingOutId: string | null;
  onPick: (planId: string) => void;
  ctaLabel?: string;
}) {
  const active = plans.filter((p) => p.isActive);
  if (active.length === 0) {
    return <p className="text-sm text-muted">Belum ada paket tersedia.</p>;
  }
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {active.map((p) => (
        <Card key={p.id}>
          <CardContent className="space-y-3 p-5">
            <div>
              <p className="text-sm font-semibold text-text">{p.name}</p>
              <p className="mt-1 text-xl font-bold tabular-nums text-text">
                {formatRupiah(p.price)}
              </p>
              <p className="text-xs text-muted">per {p.periodDays} hari</p>
            </div>
            {isOwner ? (
              <Button
                className="w-full"
                loading={checkingOutId === p.id}
                disabled={checkingOutId !== null && checkingOutId !== p.id}
                onClick={() => onPick(p.id)}
              >
                {ctaLabel}
              </Button>
            ) : (
              <p className="text-xs text-muted">Hanya pemilik toko yang dapat berlangganan.</p>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
