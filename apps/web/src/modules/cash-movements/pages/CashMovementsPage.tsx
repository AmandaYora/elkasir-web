import { useMemo } from "react";
import { ArrowDownCircle, ArrowUpCircle, Scale } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "@/shared/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/shared/components/ui/table";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { formatIDR, formatDateTime } from "@/shared/lib/formatter";
import { useAsync } from "@/shared/hooks/useAsync";
import { cashMovementsService } from "@/modules/cash-movements/services/cash-movements.service";
import { CashMovementTypeBadge } from "@/modules/cash-movements/components/CashMovementTypeBadge";
import type { CashMovement } from "@/modules/cash-movements/types/cash-movement.types";

// Nilai bertanda: capital selalu masuk (+); expense selalu keluar (-); adjustment ikut tanda amount.
const signedAmount = (m: CashMovement) => {
  if (m.type === "capital") return Math.abs(m.amount);
  if (m.type === "expense") return -Math.abs(m.amount);
  return m.amount;
};

// Halaman pemantauan (read-only): pencatatan mutasi kas dilakukan supervisor di aplikasi POS,
// bukan dari dashboard admin (operasi laci milik kasir).
export default function CashMovementsPage() {
  const movementsQuery = useAsync(() => cashMovementsService.list({ limit: 200 }), []);
  const items = movementsQuery.data?.data ?? [];

  const { cashIn, cashOut, net } = useMemo(() => {
    let inSum = 0;
    let outSum = 0;
    for (const m of items) {
      const s = signedAmount(m);
      if (s >= 0) inSum += s;
      else outSum += Math.abs(s);
    }
    return { cashIn: inSum, cashOut: outSum, net: inSum - outSum };
  }, [items]);

  const refresh = () => movementsQuery.refetch();

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div>
        <h2 className="text-lg font-semibold text-text">Mutasi Kas</h2>
        <p className="text-sm text-muted">
          Pantau setiap kas masuk dan keluar dari laci. Pencatatan dilakukan kasir/supervisor di
          POS.
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <StatCard
          label="Kas Masuk"
          value={formatIDR(cashIn)}
          hint="periode ini"
          icon={ArrowDownCircle}
          tone="text-success"
        />
        <StatCard
          label="Kas Keluar"
          value={formatIDR(cashOut)}
          hint="periode ini"
          icon={ArrowUpCircle}
          tone="text-warning"
        />
        <StatCard
          label="Mutasi Bersih"
          value={(net >= 0 ? "+" : "") + formatIDR(net)}
          hint="arus bersih"
          icon={Scale}
          tone={net >= 0 ? "text-primary" : "text-warning"}
        />
      </div>

      <Card className="overflow-hidden">
        <CardHeader>
          <CardTitle>Riwayat Mutasi</CardTitle>
          <CardDescription>Seluruh aktivitas kas fisik di laci</CardDescription>
        </CardHeader>
        {movementsQuery.loading ? (
          <LoadingState />
        ) : movementsQuery.error ? (
          <ErrorState message="Gagal memuat mutasi kas. Coba lagi." onRetry={refresh} />
        ) : items.length === 0 ? (
          <EmptyState title="Belum ada mutasi kas." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tanggal</TableHead>
                <TableHead>Jenis</TableHead>
                <TableHead>Catatan</TableHead>
                <TableHead>Oleh</TableHead>
                <TableHead>Disetujui</TableHead>
                <TableHead className="text-right">Nominal</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((m) => {
                const signed = signedAmount(m);
                return (
                  <TableRow key={m.id}>
                    <TableCell className="text-sm text-muted">
                      {formatDateTime(m.createdAt)}
                    </TableCell>
                    <TableCell>
                      <CashMovementTypeBadge type={m.type} />
                    </TableCell>
                    <TableCell className="text-sm">{m.notes ?? "—"}</TableCell>
                    <TableCell className="text-sm">{m.createdBy ?? "—"}</TableCell>
                    <TableCell className="text-sm text-muted">{m.approvedBy ?? "—"}</TableCell>
                    <TableCell
                      className={`text-right text-sm font-semibold ${signed >= 0 ? "text-success" : "text-danger"}`}
                    >
                      {signed > 0 ? "+" : ""}
                      {formatIDR(signed)}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
}

function StatCard({
  label,
  value,
  hint,
  icon: Icon,
  tone,
}: {
  label: string;
  value: string;
  hint: string;
  icon: typeof Scale;
  tone: string;
}) {
  return (
    <Card>
      <CardContent className="flex items-start justify-between p-4">
        <div>
          <div className="text-sm text-muted">{label}</div>
          <div className={`mt-1 text-2xl font-semibold ${tone}`}>{value}</div>
          <div className="mt-1 text-xs text-muted">{hint}</div>
        </div>
        <Icon className={`h-5 w-5 ${tone}`} />
      </CardContent>
    </Card>
  );
}
