import { useMemo, useState } from "react";
import { Search, CreditCard, Banknote } from "lucide-react";
import { Input } from "@/shared/components/ui/input";
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
import { Drawer } from "@/shared/components/ui/drawer";
import { Pagination } from "@/shared/components/ui/pagination";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { formatIDR, formatDateTime } from "@/shared/lib/formatter";
import { useAsync } from "@/shared/hooks/useAsync";
import { transactionsService } from "@/modules/transactions/services/transactions.service";
import { TransactionStatusBadge } from "@/modules/transactions/components/TransactionStatusBadge";
import type { Transaction } from "@/modules/transactions/types/transaction.types";

const PAGE_SIZE = 12;

const PAY_LABEL: Record<string, string> = { cash: "Tunai", qris: "QRIS" };
const ORDER_TYPE_LABEL: Record<string, string> = {
  dineIn: "Makan di Tempat",
  takeaway: "Bawa Pulang",
  pickup: "Ambil Sendiri",
  delivery: "Antar",
};
const SOURCE_LABEL: Record<string, string> = {
  cashier: "Kasir",
  self_order: "Pesan mandiri",
};

export default function TransactionsPage() {
  const transactionsQuery = useAsync(() => transactionsService.list({ limit: 200 }), []);
  const items = transactionsQuery.data?.data ?? [];

  const [q, setQ] = useState("");
  const [status, setStatus] = useState("all");
  const [method, setMethod] = useState("all");
  const [source, setSource] = useState("all");
  const [page, setPage] = useState(1);
  const [detailId, setDetailId] = useState<string | null>(null);

  const filtered = useMemo(
    () =>
      items.filter(
        (t) =>
          (status === "all" || t.status === status) &&
          (method === "all" || t.paymentMethod === method) &&
          (source === "all" || t.source === source) &&
          (q === "" ||
            t.code.toLowerCase().includes(q.toLowerCase()) ||
            t.orderType.toLowerCase().includes(q.toLowerCase()) ||
            t.paymentMethod.toLowerCase().includes(q.toLowerCase())),
      ),
    [items, q, status, method, source],
  );

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  // Jaga halaman tetap valid bila filter menciutkan hasil ke lebih sedikit halaman.
  const current = Math.min(page, totalPages);
  const paged = filtered.slice((current - 1) * PAGE_SIZE, current * PAGE_SIZE);

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div>
        <h2 className="text-lg font-semibold text-text">Riwayat Transaksi</h2>
        <p className="text-sm text-muted">{items.length} transaksi</p>
      </div>

      <Card>
        <CardContent className="p-4">
          <div className="flex flex-wrap items-center gap-2">
            <div className="relative min-w-[240px] flex-1">
              <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted" />
              <Input
                value={q}
                onChange={(e) => {
                  setQ(e.target.value);
                  setPage(1);
                }}
                placeholder="Cari kode, jenis pesanan, atau metode"
                className="pl-9"
              />
            </div>
            <Select
              value={status}
              onChange={(e) => {
                setStatus(e.target.value);
                setPage(1);
              }}
              className="w-[140px]"
            >
              <option value="all">Semua status</option>
              <option value="completed">Selesai</option>
            </Select>
            <Select
              value={method}
              onChange={(e) => {
                setMethod(e.target.value);
                setPage(1);
              }}
              className="w-[140px]"
            >
              <option value="all">Semua metode</option>
              <option value="cash">Tunai</option>
              <option value="qris">QRIS</option>
            </Select>
            <Select
              value={source}
              onChange={(e) => {
                setSource(e.target.value);
                setPage(1);
              }}
              className="w-[140px]"
            >
              <option value="all">Semua sumber</option>
              <option value="cashier">Kasir</option>
              <option value="self_order">Pesan mandiri</option>
            </Select>
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        {transactionsQuery.loading ? (
          <LoadingState />
        ) : transactionsQuery.error ? (
          <ErrorState
            message={`Gagal memuat transaksi. ${transactionsQuery.error}`}
            onRetry={() => transactionsQuery.refetch()}
          />
        ) : paged.length === 0 ? (
          <EmptyState
            title="Transaksi tidak ditemukan."
            description="Coba ubah filter atau kata kunci pencarian."
          />
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Transaksi</TableHead>
                  <TableHead>Tanggal</TableHead>
                  <TableHead>Pesanan</TableHead>
                  <TableHead>Metode</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Total</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paged.map((t) => (
                  <TableRow key={t.id} className="cursor-pointer" onClick={() => setDetailId(t.id)}>
                    <TableCell className="font-mono text-xs font-medium">{t.code}</TableCell>
                    <TableCell className="text-sm text-muted">
                      {formatDateTime(t.createdAt)}
                    </TableCell>
                    <TableCell className="text-sm">
                      <div className="font-medium">
                        {ORDER_TYPE_LABEL[t.orderType] ?? t.orderType}
                      </div>
                      <div className="text-xs text-muted">{SOURCE_LABEL[t.source] ?? t.source}</div>
                    </TableCell>
                    <TableCell>
                      <div className="inline-flex items-center gap-1.5 text-sm">
                        {t.paymentMethod === "cash" ? (
                          <Banknote className="h-3.5 w-3.5 text-muted" />
                        ) : (
                          <CreditCard className="h-3.5 w-3.5 text-muted" />
                        )}
                        {PAY_LABEL[t.paymentMethod] ?? t.paymentMethod}
                      </div>
                    </TableCell>
                    <TableCell>
                      <TransactionStatusBadge status={t.status} />
                    </TableCell>
                    <TableCell className="text-right text-sm font-semibold">
                      {formatIDR(t.total)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pagination
              page={current}
              totalPages={totalPages}
              total={filtered.length}
              onPageChange={setPage}
              label={`Menampilkan ${filtered.length === 0 ? 0 : (current - 1) * PAGE_SIZE + 1}-${Math.min(
                current * PAGE_SIZE,
                filtered.length,
              )} dari ${filtered.length}`}
            />
          </>
        )}
      </Card>

      <Drawer
        open={!!detailId}
        onClose={() => setDetailId(null)}
        title="Detail Transaksi"
        description="Rincian item dan pembayaran"
      >
        {detailId && <TransactionDetail id={detailId} />}
      </Drawer>
    </div>
  );
}

function TransactionDetail({ id }: { id: string }) {
  const detailQuery = useAsync(() => transactionsService.get(id), [id]);

  if (detailQuery.loading) return <LoadingState />;
  if (detailQuery.error)
    return (
      <ErrorState
        message={`Gagal memuat detail transaksi. ${detailQuery.error}`}
        onRetry={() => detailQuery.refetch()}
      />
    );
  const detail = detailQuery.data;
  if (!detail) return null;

  return <TransactionDetailContent detail={detail} />;
}

function TransactionDetailContent({ detail }: { detail: Transaction }) {
  return (
    <div className="space-y-5">
      <div>
        <div className="font-mono text-base font-semibold text-text">{detail.code}</div>
        <div className="text-sm text-muted">{formatDateTime(detail.createdAt)}</div>
      </div>

      <div className="flex items-center justify-between">
        <TransactionStatusBadge status={detail.status} />
        <span className="text-2xl font-semibold tracking-tight">{formatIDR(detail.total)}</span>
      </div>

      <div className="rounded-xl border border-border bg-surface-muted">
        <div className="border-b border-border px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">
          Item
        </div>
        <div className="divide-y divide-border">
          {detail.items.map((it, i) => (
            <div key={i} className="flex items-center justify-between px-4 py-3 text-sm">
              <div>
                <div className="font-medium">{it.productName}</div>
                <div className="text-xs text-muted">
                  {it.quantity} x {formatIDR(it.price)}
                  {it.note && <span className="text-warning"> ({it.note})</span>}
                </div>
              </div>
              <div className="font-medium">{formatIDR(it.lineTotal)}</div>
            </div>
          ))}
        </div>
      </div>

      <div className="space-y-3 rounded-xl border border-border p-4">
        <div className="text-xs font-medium uppercase tracking-wider text-muted">Pembayaran</div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted">Metode</span>
          <span className="font-medium">
            {PAY_LABEL[detail.paymentMethod] ?? detail.paymentMethod}
          </span>
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted">Sumber</span>
          <span className="font-medium">{SOURCE_LABEL[detail.source] ?? detail.source}</span>
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted">Jenis Pesanan</span>
          <span className="font-medium">
            {ORDER_TYPE_LABEL[detail.orderType] ?? detail.orderType}
          </span>
        </div>
        {detail.customerNote && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted">Catatan</span>
            <span className="font-medium">{detail.customerNote}</span>
          </div>
        )}
        <div className="flex items-center justify-between border-t border-border pt-3 text-sm">
          <span className="text-muted">Subtotal</span>
          <span>{formatIDR(detail.subtotal)}</span>
        </div>
        {detail.discount > 0 && (
          <div className="flex items-center justify-between text-sm text-success">
            <span className="text-muted">Diskon</span>
            <span>-{formatIDR(detail.discount)}</span>
          </div>
        )}
        {detail.tax > 0 && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted">Pajak</span>
            <span>{formatIDR(detail.tax)}</span>
          </div>
        )}
        <div className="flex items-center justify-between text-sm font-semibold">
          <span>Total Dibayar</span>
          <span>{formatIDR(detail.total)}</span>
        </div>
        {detail.paymentMethod === "cash" && (
          <>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted">Diterima</span>
              <span>{formatIDR(detail.amountReceived)}</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted">Kembalian</span>
              <span>{formatIDR(detail.changeAmount)}</span>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
