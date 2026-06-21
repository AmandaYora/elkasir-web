import { useMemo, useRef, useState } from "react";
import { QRCodeCanvas } from "qrcode.react";
import { Search, Plus, QrCode, Printer, Pencil } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
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
import { Modal } from "@/shared/components/ui/modal";
import { Drawer } from "@/shared/components/ui/drawer";
import { Pagination } from "@/shared/components/ui/pagination";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { useAsync } from "@/shared/hooks/useAsync";
import { tablesService } from "@/modules/tables/services/tables.service";
import { tableSchema } from "@/modules/tables/schemas/table.schema";
import { TableStatusBadge } from "@/modules/tables/components/TableStatusBadge";
import type { DiningTable, TableInput, TableStatus } from "@/modules/tables/types/table.types";

const PAGE_SIZE = 10;

const orderUrl = (code: string) => `${window.location.origin}/order/${code}`;

export default function TablesPage() {
  const tablesQuery = useAsync(() => tablesService.list({ limit: 200 }), []);
  const list = tablesQuery.data?.data ?? [];

  const [q, setQ] = useState("");
  const [status, setStatus] = useState("all");
  const [page, setPage] = useState(1);
  const [detail, setDetail] = useState<DiningTable | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<DiningTable | null>(null);
  const [busy, setBusy] = useState(false);
  const qrRef = useRef<HTMLCanvasElement | null>(null);

  const filtered = useMemo(
    () =>
      list.filter(
        (t) =>
          (status === "all" || t.status === status) &&
          (q === "" ||
            t.code.toLowerCase().includes(q.toLowerCase()) ||
            t.name.toLowerCase().includes(q.toLowerCase()) ||
            t.area.toLowerCase().includes(q.toLowerCase())),
      ),
    [list, q, status],
  );

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  const refresh = () => tablesQuery.refetch();

  const submit = async (body: TableInput) => {
    setBusy(true);
    try {
      if (editing) {
        const updated = await tablesService.update(editing.id, body);
        setDetail((d) => (d && d.id === updated.id ? updated : d));
        toast.success("Meja berhasil diperbarui");
      } else {
        await tablesService.create(body);
        toast.success("Meja berhasil ditambahkan");
      }
      setFormOpen(false);
      setEditing(null);
      refresh();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Gagal menyimpan meja");
    } finally {
      setBusy(false);
    }
  };

  const toggleStatus = async (t: DiningTable) => {
    const next: TableStatus = t.status === "active" ? "inactive" : "active";
    setBusy(true);
    try {
      const updated = await tablesService.update(t.id, {
        code: t.code,
        name: t.name,
        area: t.area,
        seats: t.seats,
        status: next,
      });
      setDetail((d) => (d && d.id === updated.id ? updated : d));
      toast.success("Meja berhasil diperbarui");
      refresh();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Gagal memperbarui meja");
    } finally {
      setBusy(false);
    }
  };

  const printQr = (t: DiningTable) => {
    const canvas = qrRef.current;
    const dataUrl = canvas ? canvas.toDataURL("image/png") : "";
    const win = window.open("", "_blank", "width=420,height=560");
    if (!win) {
      toast.error("Popup diblokir browser");
      return;
    }
    win.document.write(`<!doctype html><html><head><title>QR Meja ${t.name}</title>
      <style>
        body{font-family:Inter,system-ui,sans-serif;text-align:center;padding:32px;color:#0f172a}
        h1{font-size:22px;margin:0 0 4px} p{color:#64748b;margin:0 0 18px;font-size:13px}
        img{width:300px;height:300px} .code{margin-top:12px;font-weight:700;letter-spacing:1px}
      </style></head><body>
      <h1>Scan untuk Pesan</h1>
      <p>Meja ${t.name} &middot; ${t.area}</p>
      <img src="${dataUrl}" alt="QR Meja ${t.name}" />
      <div class="code">MEJA ${t.code}</div>
      <script>window.onload=function(){window.print();}</script>
      </body></html>`);
    win.document.close();
  };

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">Kelola Meja</h2>
          <p className="text-sm text-muted">
            {list.length} meja · QR dibuat dari kode meja untuk dicetak & ditempel
          </p>
        </div>
        <Button
          size="sm"
          onClick={() => {
            setEditing(null);
            setFormOpen(true);
          }}
        >
          <Plus className="h-4 w-4" /> Tambah Meja
        </Button>
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
                placeholder="Cari kode, nama, atau area…"
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
              <option value="active">Aktif</option>
              <option value="inactive">Nonaktif</option>
            </Select>
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        {tablesQuery.loading ? (
          <LoadingState />
        ) : tablesQuery.error ? (
          <ErrorState message={tablesQuery.error} onRetry={refresh} />
        ) : paged.length === 0 ? (
          <EmptyState title="Belum ada meja" description="Tambahkan meja pertama Anda." />
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Kode</TableHead>
                  <TableHead>Nama</TableHead>
                  <TableHead>Area</TableHead>
                  <TableHead className="text-right">Kursi</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">QR</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paged.map((t) => (
                  <TableRow key={t.id} className="cursor-pointer" onClick={() => setDetail(t)}>
                    <TableCell className="font-mono text-sm font-medium">{t.code}</TableCell>
                    <TableCell className="text-sm">{t.name}</TableCell>
                    <TableCell className="text-sm text-muted">{t.area}</TableCell>
                    <TableCell className="text-right text-sm">{t.seats}</TableCell>
                    <TableCell>
                      <TableStatusBadge status={t.status} />
                    </TableCell>
                    <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                      <Button variant="outline" size="sm" onClick={() => setDetail(t)}>
                        <QrCode className="h-3.5 w-3.5" /> Lihat
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pagination
              page={page}
              totalPages={totalPages}
              total={filtered.length}
              onPageChange={setPage}
              label={`Menampilkan ${paged.length} dari ${filtered.length} meja`}
            />
          </>
        )}
      </Card>

      {/* QR detail drawer */}
      <Drawer
        open={!!detail}
        onClose={() => setDetail(null)}
        title={detail ? `Meja ${detail.name}` : undefined}
        description={detail ? `${detail.area} · ${detail.seats} kursi` : undefined}
      >
        {detail && (
          <div className="space-y-5">
            <div className="flex flex-col items-center rounded-xl border border-border p-5">
              <QRCodeCanvas ref={qrRef} value={orderUrl(detail.code)} size={220} includeMargin />
              <div className="mt-3 font-mono text-sm font-semibold tracking-wider">
                MEJA {detail.code}
              </div>
              <div className="mt-1 break-all text-center text-xs text-muted">
                {orderUrl(detail.code)}
              </div>
            </div>
            <div className="flex gap-2">
              <Button className="flex-1" onClick={() => printQr(detail)}>
                <Printer className="h-4 w-4" /> Cetak QR
              </Button>
              <Button
                variant="outline"
                className="flex-1"
                loading={busy}
                onClick={() => toggleStatus(detail)}
              >
                {detail.status === "active" ? "Nonaktifkan" : "Aktifkan"}
              </Button>
            </div>
            <Button
              variant="secondary"
              className="w-full"
              onClick={() => {
                setEditing(detail);
                setDetail(null);
                setFormOpen(true);
              }}
            >
              <Pencil className="h-3.5 w-3.5" /> Ubah Meja
            </Button>
            <p className="text-center text-xs text-muted">
              Cetak lalu tempel di meja. Pelanggan scan → memilih menu → bayar QRIS → pesanan masuk
              ke kasir.
            </p>
          </div>
        )}
      </Drawer>

      {/* Create / edit form */}
      <Modal
        open={formOpen}
        onClose={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        title={editing ? "Ubah Meja" : "Tambah Meja"}
        description="Kode meja menjadi isi QR (mis. A1 → /order/A1). Harus unik."
      >
        <TableForm
          key={editing?.id ?? "new"}
          editing={editing}
          existing={list}
          busy={busy}
          onSubmit={submit}
        />
      </Modal>
    </div>
  );
}

function TableForm({
  editing,
  existing,
  busy,
  onSubmit,
}: {
  editing: DiningTable | null;
  existing: DiningTable[];
  busy: boolean;
  onSubmit: (body: TableInput) => void;
}) {
  const [form, setForm] = useState({
    code: editing?.code ?? "",
    name: editing?.name ?? "",
    area: editing?.area ?? "Indoor",
    seats: String(editing?.seats ?? 2),
    status: (editing?.status ?? "active") as TableStatus,
  });

  const submit = () => {
    const code = form.code.trim();
    const name = form.name.trim() || code;
    const body: TableInput = {
      code,
      name,
      area: form.area,
      seats: Number(form.seats) || 0,
      status: form.status,
    };
    const parsed = tableSchema.safeParse(body);
    if (!parsed.success) {
      toast.error(parsed.error.issues[0]?.message ?? "Periksa input.");
      return;
    }
    if (existing.some((t) => t.id !== editing?.id && t.code.toLowerCase() === code.toLowerCase())) {
      toast.error("Kode meja sudah dipakai");
      return;
    }
    onSubmit(body);
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label>Kode Meja</Label>
        <Input
          value={form.code}
          onChange={(e) => setForm({ ...form, code: e.target.value })}
          placeholder="mis. A3"
        />
      </div>
      <div className="grid gap-2">
        <Label>Nama Tampilan</Label>
        <Input
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          placeholder="mis. A3 (opsional, default = kode)"
        />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="grid gap-2">
          <Label>Area</Label>
          <Select value={form.area} onChange={(e) => setForm({ ...form, area: e.target.value })}>
            <option value="Indoor">Indoor</option>
            <option value="Outdoor">Outdoor</option>
            <option value="Private">Private</option>
          </Select>
        </div>
        <div className="grid gap-2">
          <Label>Kursi</Label>
          <Input
            type="number"
            value={form.seats}
            onChange={(e) => setForm({ ...form, seats: e.target.value })}
          />
        </div>
      </div>
      <div className="grid gap-2">
        <Label>Status</Label>
        <Select
          value={form.status}
          onChange={(e) => setForm({ ...form, status: e.target.value as TableStatus })}
        >
          <option value="active">Aktif</option>
          <option value="inactive">Nonaktif</option>
        </Select>
      </div>
      <div className="flex justify-end">
        <Button loading={busy} onClick={submit}>
          {editing ? "Simpan Perubahan" : "Simpan Meja"}
        </Button>
      </div>
    </div>
  );
}
