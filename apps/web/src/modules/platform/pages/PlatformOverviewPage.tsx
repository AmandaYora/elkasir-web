import { CreditCard, Wallet, ShieldCheck } from "lucide-react";
import { StatCard } from "@/modules/dashboard/components/StatCard";
import { LoadingState, ErrorState } from "@/shared/components/feedback";
import { useAsync } from "@/shared/hooks/useAsync";
import { platformService } from "@/modules/platform/services/platform.service";

const formatRupiah = (n: number) => `Rp ${n.toLocaleString("id-ID")}`;

// Ringkasan — the reconciliation dashboard (§2.5): subscription revenue + tenants' unwithdrawn
// QRIS balance, which together should equal the real Tripay/Midtrans gateway balance (manual
// sanity check only — not automated reconciliation, §5).
export default function PlatformOverviewPage() {
  const revenueQuery = useAsync(() => platformService.revenue(), []);
  const revenue = revenueQuery.data;

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div>
        <h2 className="text-lg font-semibold text-text">Ringkasan</h2>
        <p className="text-sm text-muted">Dasbor rekonsiliasi lintas-tenant Konsol Platform.</p>
      </div>

      {revenueQuery.loading ? (
        <LoadingState />
      ) : revenueQuery.error || !revenue ? (
        <ErrorState message="Gagal memuat ringkasan. Coba lagi." onRetry={revenueQuery.refetch} />
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-3">
            <StatCard
              label="Pendapatan Langganan"
              value={formatRupiah(revenue.subscriptionRevenue)}
              icon={CreditCard}
              accent="primary"
            />
            <StatCard
              label="Saldo Tenant Belum Dicairkan"
              value={formatRupiah(revenue.tenantAvailableBalance)}
              icon={Wallet}
              accent="warning"
            />
            <StatCard
              label="Total Termonitor"
              value={formatRupiah(revenue.totalMonitored)}
              icon={ShieldCheck}
              accent="success"
            />
          </div>
          <p className="text-xs text-muted">
            Total Termonitor = Pendapatan Langganan + Saldo Tenant Belum Dicairkan — seharusnya sama
            dengan saldo asli di Tripay/Midtrans (pengecekan manual, bukan rekonsiliasi otomatis).
          </p>
        </>
      )}
    </div>
  );
}
