import { Building2 } from "lucide-react";
import { Card } from "@/shared/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/shared/components/ui/table";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { useAsync } from "@/shared/hooks/useAsync";
import { platformService } from "@/modules/platform/services/platform.service";

const formatRupiah = (n: number) => `Rp ${n.toLocaleString("id-ID")}`;

// Revenue Tenant — read-only, sorted by balance descending (server-side, §2.6). A
// reconciliation view, not an action page — no edit/claim affordances here.
export default function PlatformTenantRevenuePage() {
  const revenueQuery = useAsync(() => platformService.tenantsRevenue(), []);
  const list = revenueQuery.data ?? [];

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div>
        <h2 className="text-lg font-semibold text-text">Revenue Tenant</h2>
        <p className="text-sm text-muted">
          Saldo QRIS self-order yang belum dicairkan, per tenant — urut terbesar.
        </p>
      </div>

      <Card className="overflow-hidden">
        {revenueQuery.loading ? (
          <LoadingState />
        ) : revenueQuery.error ? (
          <ErrorState message="Gagal memuat data. Coba lagi." onRetry={revenueQuery.refetch} />
        ) : list.length === 0 ? (
          <EmptyState title="Belum ada data" description="Belum ada tenant dengan saldo QRIS." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tenant</TableHead>
                <TableHead>Saldo Belum Dicairkan</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((t) => (
                <TableRow key={t.storeId}>
                  <TableCell>
                    <div className="flex items-center gap-2.5">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                        <Building2 className="h-4 w-4" />
                      </div>
                      <div>
                        <p className="text-sm font-medium">{t.name}</p>
                        <p className="text-xs text-muted">{t.slug}</p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell className="text-sm font-semibold tabular-nums">
                    {formatRupiah(t.balance)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
}
