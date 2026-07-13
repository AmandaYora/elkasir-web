import { useState } from "react";
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
import { Pagination } from "@/shared/components/ui/pagination";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { useAsync } from "@/shared/hooks/useAsync";
import { platformService } from "@/modules/platform/services/platform.service";
import { PlatformWithdrawalStatusBadge } from "@/modules/platform/components/PlatformWithdrawalStatusBadge";

const formatRupiah = (n: number) => `Rp ${n.toLocaleString("id-ID")}`;
const formatDateTime = (s?: string) => (s ? new Date(s).toLocaleString("id-ID") : "—");

const LIMIT = 20;

// Riwayat Penarikan — the full audit trail (§2.8), any status, server-paginated (unlike this
// app's other list pages, which paginate client-side — this one genuinely needs it since the
// history can grow unbounded across every tenant).
export default function PlatformWithdrawalHistoryPage() {
  const [page, setPage] = useState(1);
  const historyQuery = useAsync(
    () => platformService.withdrawalHistory({ page, limit: LIMIT }),
    [page],
  );

  const rows = historyQuery.data?.data ?? [];
  const meta = historyQuery.data?.meta;

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div>
        <h2 className="text-lg font-semibold text-text">Riwayat Penarikan</h2>
        <p className="text-sm text-muted">
          Jejak audit seluruh permintaan pencairan, semua status.
        </p>
      </div>

      <Card className="overflow-hidden">
        {historyQuery.loading ? (
          <LoadingState />
        ) : historyQuery.error ? (
          <ErrorState message="Gagal memuat riwayat. Coba lagi." onRetry={historyQuery.refetch} />
        ) : rows.length === 0 ? (
          <EmptyState title="Belum ada riwayat" description="Belum ada permintaan pencairan." />
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Tenant</TableHead>
                  <TableHead>Jumlah</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Diajukan</TableHead>
                  <TableHead>Diklaim</TableHead>
                  <TableHead>Diselesaikan</TableHead>
                  <TableHead>Alasan</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((w) => (
                  <TableRow key={w.id}>
                    <TableCell>
                      <div className="flex items-center gap-2.5">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                          <Building2 className="h-4 w-4" />
                        </div>
                        <span className="text-sm font-medium">{w.tenantName}</span>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm font-semibold tabular-nums">
                      {formatRupiah(w.amount)}
                    </TableCell>
                    <TableCell>
                      <PlatformWithdrawalStatusBadge status={w.status} />
                    </TableCell>
                    <TableCell className="text-sm text-muted">
                      {formatDateTime(w.createdAt)}
                    </TableCell>
                    <TableCell className="text-sm text-muted">
                      {w.claimedAt
                        ? `${formatDateTime(w.claimedAt)} · ${w.claimantName ?? "—"}`
                        : "—"}
                    </TableCell>
                    <TableCell className="text-sm text-muted">
                      {formatDateTime(w.processedAt)}
                    </TableCell>
                    <TableCell className="max-w-[200px] truncate text-sm text-muted">
                      {w.rejectedReason || "—"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {meta && (
              <Pagination
                page={meta.page}
                totalPages={meta.total_pages}
                total={meta.total}
                onPageChange={setPage}
                label={`Menampilkan ${rows.length} dari ${meta.total} riwayat`}
              />
            )}
          </>
        )}
      </Card>
    </div>
  );
}
