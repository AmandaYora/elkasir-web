import { useMemo, useState } from "react";
import { toast } from "sonner";
import { QrCode, Banknote, ChefHat, ArrowRight, Loader2 } from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { Badge } from "@/shared/components/ui/badge";
import { Select } from "@/shared/components/ui/select";
import { Card, CardContent } from "@/shared/components/ui/card";
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
import { cn } from "@/shared/lib/cn";
import { selfOrderService } from "@/modules/self-order/services/self-order.service";
import {
  OrderStageBadge,
  PaymentStatusBadge,
} from "@/modules/self-order/components/SelfOrderBadges";
import {
  ORDER_STAGE_LABEL,
  type SelfOrder,
  type SelfOrderStatus,
} from "@/modules/self-order/types/self-order.types";

type Stage = SelfOrderStatus;

const NEXT_STAGE: Record<Stage, Stage | null> = {
  placed: "preparing",
  preparing: "completed",
  completed: null,
};

const itemSummary = (o: SelfOrder) =>
  o.items.length === 1
    ? `${o.items[0].quantity}× ${o.items[0].productName}`
    : `${o.items[0].quantity}× ${o.items[0].productName} +${o.items.length - 1} lainnya`;

// Layar staf "Pesanan Masuk": lihat pesanan masuk self-order, tebus kode klaim untuk
// pembayaran tunai di kasir, dan majukan tahap penyiapan.
export default function IncomingOrdersPage() {
  const ordersQuery = useAsync(() => selfOrderService.list({ limit: 200 }), []);
  const list = ordersQuery.data?.data ?? [];

  const [filter, setFilter] = useState("all");
  const [advancingId, setAdvancingId] = useState<string | null>(null);

  const refresh = () => ordersQuery.refetch();

  const filtered = useMemo(() => {
    if (filter === "all") return list;
    if (filter === "unpaid") return list.filter((o) => o.paymentStatus === "unpaid");
    return list.filter((o) => o.status === filter);
  }, [list, filter]);

  const unpaidCount = list.filter((o) => o.paymentStatus === "unpaid").length;

  // Majukan tahap penyiapan (placed → preparing → completed).
  const advance = async (id: string, status: Stage) => {
    setAdvancingId(id);
    try {
      const updated = await selfOrderService.updateStatus(id, status);
      toast.success(`Pesanan Meja ${updated.tableName} → ${ORDER_STAGE_LABEL[updated.status]}`);
      refresh();
    } catch (e) {
      toast.error("Gagal memperbarui pesanan. Coba lagi.");
    } finally {
      setAdvancingId(null);
    }
  };

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div>
        <h2 className="text-lg font-semibold text-text">Pesanan Masuk</h2>
        <p className="text-sm text-muted">
          {list.length} pesanan mandiri · {unpaidCount} belum dibayar di kasir
        </p>
      </div>

      <Card>
        <CardContent className="p-4">
          <div className="flex flex-wrap items-center gap-2">
            <Select
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="w-[180px]"
            >
              <option value="all">Semua pesanan</option>
              <option value="unpaid">Belum bayar (di kasir)</option>
              <option value="placed">Masuk</option>
              <option value="preparing">Disiapkan</option>
              <option value="completed">Selesai</option>
            </Select>
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        {ordersQuery.loading ? (
          <LoadingState />
        ) : ordersQuery.error ? (
          <ErrorState message="Gagal memuat pesanan. Coba lagi." onRetry={refresh} />
        ) : filtered.length === 0 ? (
          <EmptyState title="Tidak ada pesanan" description="Tidak ada pesanan pada filter ini." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                {["Waktu", "Meja", "Pesanan", "Total", "Pembayaran", "Tahap", ""].map((h, i) => (
                  <TableHead
                    key={h || i}
                    className={cn(
                      "text-xs uppercase tracking-wider text-muted",
                      (h === "Total" || h === "") && "text-right",
                    )}
                  >
                    {h}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((o) => {
                const next = NEXT_STAGE[o.status];
                return (
                  <TableRow key={o.id}>
                    <TableCell className="whitespace-nowrap text-sm text-muted">
                      {formatDateTime(o.createdAt)}
                    </TableCell>
                    <TableCell className="font-mono text-sm font-medium">{o.tableName}</TableCell>
                    <TableCell className="max-w-[220px] text-sm">
                      <span className="block truncate">{itemSummary(o)}</span>
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-right text-sm font-semibold">
                      {formatIDR(o.total)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Badge
                          tone="neutral"
                          className="gap-1"
                          title={o.paymentMethod === "qris" ? "QRIS" : "Tunai di kasir"}
                        >
                          {o.paymentMethod === "qris" ? (
                            <QrCode className="h-3 w-3" />
                          ) : (
                            <Banknote className="h-3 w-3" />
                          )}
                          {o.paymentMethod === "qris" ? "QRIS" : "Tunai"}
                        </Badge>
                        <PaymentStatusBadge status={o.paymentStatus} />
                      </div>
                    </TableCell>
                    <TableCell>
                      <OrderStageBadge status={o.status} />
                    </TableCell>
                    <TableCell className="text-right">
                      {next && (
                        <Button
                          variant="outline"
                          size="sm"
                          className="gap-1.5"
                          onClick={() => advance(o.id, next)}
                          disabled={o.paymentStatus !== "paid" || advancingId === o.id}
                          title={
                            o.paymentStatus !== "paid" ? "Menunggu pembayaran di kasir" : undefined
                          }
                        >
                          {advancingId === o.id ? (
                            <Loader2 className="h-3.5 w-3.5 animate-spin" />
                          ) : next === "preparing" ? (
                            <ChefHat className="h-3.5 w-3.5" />
                          ) : (
                            <ArrowRight className="h-3.5 w-3.5" />
                          )}
                          {next === "preparing" ? "Siapkan" : "Selesai"}
                        </Button>
                      )}
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
