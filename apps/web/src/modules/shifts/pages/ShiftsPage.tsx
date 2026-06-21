import { useMemo, useState } from "react";
import { Clock, Wallet, ArrowUpRight, ArrowDownRight, Minus } from "lucide-react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/shared/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/shared/components/ui/table";
import { Drawer } from "@/shared/components/ui/drawer";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { formatIDR, formatDateTime, formatDate } from "@/shared/lib/formatter";
import { useAsync } from "@/shared/hooks/useAsync";
import { shiftsService } from "@/modules/shifts/services/shifts.service";
import { ShiftStatusBadge } from "@/modules/shifts/components/ShiftStatusBadge";
import type { Shift } from "@/modules/shifts/types/shift.types";

const shortStaff = (id: string) => (id.length > 12 ? `${id.slice(0, 8)}…${id.slice(-4)}` : id);

const varianceOf = (s: Shift): number | null =>
  s.variance != null
    ? s.variance
    : s.actualCash != null && s.expectedCash != null
      ? s.actualCash - s.expectedCash
      : null;

function StatTile({
  label,
  value,
  icon: Icon,
  caption,
  tone = "text-text",
}: {
  label: string;
  value: string;
  icon: typeof Clock;
  caption: string;
  tone?: string;
}) {
  return (
    <Card>
      <CardContent className="p-4 pt-4">
        <div className="flex items-center justify-between">
          <span className="text-xs uppercase tracking-wider text-muted">{label}</span>
          <Icon className="h-4 w-4 text-muted" />
        </div>
        <div className={`mt-2 text-2xl font-semibold tracking-tight ${tone}`}>{value}</div>
        <div className="mt-1 text-xs text-muted">{caption}</div>
      </CardContent>
    </Card>
  );
}

export default function ShiftsPage() {
  const shiftsQuery = useAsync(() => shiftsService.list({ limit: 200 }), []);
  const items = shiftsQuery.data?.data ?? [];

  const [detailId, setDetailId] = useState<string | null>(null);

  const stats = useMemo(() => {
    const openShifts = items.filter((s) => s.status === "open").length;
    const totalCashSales = items.reduce((a, s) => a + s.cashSales, 0);
    const totalQrisSales = items.reduce((a, s) => a + s.qrisSales, 0);
    const totalVariance = items.reduce((a, s) => a + (varianceOf(s) ?? 0), 0);
    return { openShifts, totalCashSales, totalQrisSales, totalVariance };
  }, [items]);

  return (
    <div className="space-y-6 p-4 md:p-6">
      <div>
        <h2 className="text-lg font-semibold text-text">Shift</h2>
        <p className="text-sm text-muted">Riwayat dan rekonsiliasi shift kasir.</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatTile
          label="Shift Aktif"
          value={String(stats.openShifts)}
          icon={Clock}
          caption="sedang aktif"
          tone="text-success"
        />
        <StatTile
          label="Penjualan Tunai"
          value={formatIDR(stats.totalCashSales)}
          icon={Wallet}
          caption="seluruh shift"
          tone="text-primary"
        />
        <StatTile
          label="Penjualan QRIS"
          value={formatIDR(stats.totalQrisSales)}
          icon={ArrowUpRight}
          caption="seluruh shift"
        />
        <StatTile
          label="Total Selisih"
          value={(stats.totalVariance >= 0 ? "+" : "") + formatIDR(stats.totalVariance)}
          icon={Minus}
          caption="dari seluruh shift"
          tone={stats.totalVariance === 0 ? "text-success" : "text-warning"}
        />
      </div>

      <Card className="overflow-hidden">
        <CardHeader>
          <CardTitle>Riwayat Shift</CardTitle>
          <CardDescription>Aktivitas kasir terbaru</CardDescription>
        </CardHeader>
        {shiftsQuery.loading ? (
          <LoadingState />
        ) : shiftsQuery.error ? (
          <ErrorState
            message={`Gagal memuat shift. ${shiftsQuery.error}`}
            onRetry={() => shiftsQuery.refetch()}
          />
        ) : items.length === 0 ? (
          <EmptyState
            title="Belum ada riwayat shift."
            description="Shift kasir yang dibuka akan muncul di sini."
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Staf</TableHead>
                <TableHead>Dibuka</TableHead>
                <TableHead>Ditutup</TableHead>
                <TableHead className="text-right">Kas Awal</TableHead>
                <TableHead className="text-right">Penjualan Tunai</TableHead>
                <TableHead className="text-right">Penjualan QRIS</TableHead>
                <TableHead className="text-right">Selisih</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((s) => {
                const diff = varianceOf(s);
                return (
                  <TableRow key={s.id} className="cursor-pointer" onClick={() => setDetailId(s.id)}>
                    <TableCell className="font-mono text-sm" title={s.staffId}>
                      {shortStaff(s.staffId)}
                    </TableCell>
                    <TableCell className="text-sm text-muted">
                      {formatDateTime(s.openedAt)}
                    </TableCell>
                    <TableCell className="text-sm text-muted">
                      {s.closedAt ? formatDateTime(s.closedAt) : "—"}
                    </TableCell>
                    <TableCell className="text-right text-sm">{formatIDR(s.initialCash)}</TableCell>
                    <TableCell className="text-right text-sm">{formatIDR(s.cashSales)}</TableCell>
                    <TableCell className="text-right text-sm">{formatIDR(s.qrisSales)}</TableCell>
                    <TableCell className="text-right">
                      {diff === null ? (
                        <span className="text-xs text-muted">Aktif</span>
                      ) : diff === 0 ? (
                        <span className="inline-flex items-center gap-1 text-sm font-medium text-success">
                          Seimbang
                        </span>
                      ) : diff > 0 ? (
                        <span className="inline-flex items-center gap-1 text-sm font-medium text-warning">
                          <ArrowUpRight className="h-3 w-3" />+{formatIDR(diff)}
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 text-sm font-medium text-danger">
                          <ArrowDownRight className="h-3 w-3" />
                          {formatIDR(diff)}
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      <ShiftStatusBadge status={s.status} />
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </Card>

      <Drawer
        open={!!detailId}
        onClose={() => setDetailId(null)}
        title="Detail Shift"
        description="Rekonsiliasi kas"
      >
        {detailId && <ShiftDetail id={detailId} />}
      </Drawer>
    </div>
  );
}

function ShiftDetail({ id }: { id: string }) {
  const detailQuery = useAsync(() => shiftsService.get(id), [id]);

  if (detailQuery.loading) return <LoadingState />;
  if (detailQuery.error)
    return (
      <ErrorState
        message={`Gagal memuat detail shift. ${detailQuery.error}`}
        onRetry={() => detailQuery.refetch()}
      />
    );
  const detail = detailQuery.data;
  if (!detail) return null;

  return <ShiftDetailContent detail={detail} />;
}

function ShiftDetailContent({ detail }: { detail: Shift }) {
  const diff = varianceOf(detail);
  const rows: [string, string][] = [
    ["Staf", detail.staffId],
    ["Dibuka", formatDateTime(detail.openedAt)],
    ["Ditutup", detail.closedAt ? formatDateTime(detail.closedAt) : "Masih aktif"],
    ["Kas Awal", formatIDR(detail.initialCash)],
    ["Penjualan Tunai", formatIDR(detail.cashSales)],
    ["Penjualan QRIS", formatIDR(detail.qrisSales)],
    ["Modal Tambahan", formatIDR(detail.additionalCapital)],
    ["Biaya", formatIDR(detail.expenses)],
    ["Penarikan", formatIDR(detail.withdrawals)],
    ["Penyesuaian", formatIDR(detail.adjustments)],
    ["Perkiraan Kas", detail.expectedCash != null ? formatIDR(detail.expectedCash) : "—"],
    ["Kas Aktual", detail.actualCash != null ? formatIDR(detail.actualCash) : "—"],
    ...(detail.closeApprovedBy
      ? ([["Selisih disetujui oleh", detail.closeApprovedBy]] as [string, string][])
      : []),
  ];

  return (
    <div className="space-y-4">
      <div>
        <div className="text-base font-semibold text-text">Detail Shift</div>
        <div className="font-mono text-xs text-muted">
          {shortStaff(detail.staffId)} · {formatDate(detail.openedAt)}
        </div>
      </div>

      <div
        className={`rounded-xl border p-5 text-center ${
          diff === null
            ? "border-border bg-surface-muted"
            : diff === 0
              ? "border-success/20 bg-success-soft"
              : diff > 0
                ? "border-warning/30 bg-warning-soft"
                : "border-danger/20 bg-danger-soft"
        }`}
      >
        <div className="text-xs uppercase tracking-wider text-muted">Selisih</div>
        <div className="mt-1 text-3xl font-semibold tracking-tight">
          {diff === null ? "—" : (diff > 0 ? "+" : "") + formatIDR(diff)}
        </div>
        <div className="mt-1 text-xs text-muted">
          {diff === null
            ? "Shift sedang berjalan"
            : diff === 0
              ? "Seimbang sempurna"
              : "Terdeteksi selisih kas"}
        </div>
      </div>

      <div className="rounded-xl border border-border">
        {rows.map(([k, v], i) => (
          <div
            key={k}
            className={`flex items-center justify-between px-4 py-3 text-sm ${i ? "border-t border-border" : ""}`}
          >
            <span className="text-muted">{k}</span>
            <span className="font-medium">{v}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
