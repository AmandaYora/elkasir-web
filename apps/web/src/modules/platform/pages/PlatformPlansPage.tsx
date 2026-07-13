import { useState } from "react";
import { Plus, MoreHorizontal, Pencil, Layers } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
import { Checkbox } from "@/shared/components/ui/checkbox";
import { Badge } from "@/shared/components/ui/badge";
import { Card } from "@/shared/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/shared/components/ui/table";
import { Dropdown, DropdownItem } from "@/shared/components/ui/dropdown";
import { Modal } from "@/shared/components/ui/modal";
import { FieldError } from "@/shared/components/ui/field-error";
import { MoneyInput } from "@/shared/components/ui/money-input";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { useAsync } from "@/shared/hooks/useAsync";
import { zodFieldErrors } from "@/shared/lib/form";
import { platformService } from "@/modules/platform/services/platform.service";
import { planSchema } from "@/modules/platform/schemas/plan.schema";
import type { Plan, PlanInput } from "@/modules/platform/types/platform.types";

const formatRupiah = (n: number) => `Rp ${n.toLocaleString("id-ID")}`;

export default function PlatformPlansPage() {
  const plansQuery = useAsync(() => platformService.listPlans(), []);
  const list = plansQuery.data ?? [];

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Plan | null>(null);

  const refresh = () => plansQuery.refetch();

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">Paket</h2>
          <p className="text-sm text-muted">{list.length} paket langganan</p>
        </div>
        <Button
          size="sm"
          onClick={() => {
            setEditing(null);
            setFormOpen(true);
          }}
        >
          <Plus className="h-4 w-4" /> Tambah Paket
        </Button>
      </div>

      <Card className="overflow-hidden">
        {plansQuery.loading ? (
          <LoadingState />
        ) : plansQuery.error ? (
          <ErrorState message="Gagal memuat paket. Coba lagi." onRetry={refresh} />
        ) : list.length === 0 ? (
          <EmptyState title="Belum ada paket" description="Tambahkan paket langganan pertama." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Paket</TableHead>
                <TableHead>Harga</TableHead>
                <TableHead>Periode</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((p) => (
                <TableRow key={p.id}>
                  <TableCell>
                    <div className="flex items-center gap-2.5">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                        <Layers className="h-4 w-4" />
                      </div>
                      <div>
                        <p className="text-sm font-medium">{p.name}</p>
                        <p className="text-xs text-muted">{p.code}</p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell className="text-sm font-semibold tabular-nums">
                    {formatRupiah(p.price)}
                  </TableCell>
                  <TableCell className="text-sm text-muted">{p.periodDays} hari</TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1.5">
                      <Badge tone={p.isActive ? "success" : "neutral"}>
                        {p.isActive ? "Aktif" : "Nonaktif"}
                      </Badge>
                      {p.renewalOnly && (
                        <Badge tone="warning">Terkunci (khusus perpanjangan)</Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Dropdown
                      trigger={
                        <Button variant="ghost" size="icon">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      }
                    >
                      <DropdownItem
                        onClick={() => {
                          setEditing(p);
                          setFormOpen(true);
                        }}
                      >
                        <Pencil className="h-3.5 w-3.5" /> Ubah
                      </DropdownItem>
                    </Dropdown>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      <Modal
        open={formOpen}
        onClose={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        title={editing ? "Ubah Paket" : "Tambah Paket"}
        description="Paket langganan yang tersedia untuk tenant checkout di Langganan."
      >
        <PlanForm
          key={editing?.id ?? "new"}
          editing={editing}
          onDone={() => {
            setFormOpen(false);
            setEditing(null);
            refresh();
          }}
        />
      </Modal>
    </div>
  );
}

function PlanForm({ editing, onDone }: { editing: Plan | null; onDone: () => void }) {
  const [form, setForm] = useState<PlanInput>({
    code: editing?.code ?? "",
    name: editing?.name ?? "",
    price: editing?.price ?? 0,
    periodDays: editing?.periodDays ?? 30,
    isActive: editing?.isActive ?? true,
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

  const setField = <K extends keyof PlanInput>(key: K, value: PlanInput[K]) => {
    setForm((f) => ({ ...f, [key]: value }));
    setErrors((e) => (e[key] ? { ...e, [key]: "" } : e));
  };

  const submit = async () => {
    const parsed = planSchema.safeParse(form);
    if (!parsed.success) {
      setErrors(zodFieldErrors(parsed.error));
      return;
    }
    setBusy(true);
    try {
      if (editing) await platformService.updatePlan(editing.id, parsed.data);
      else await platformService.createPlan(parsed.data);
      toast.success(editing ? "Paket berhasil diperbarui" : "Paket berhasil ditambahkan");
      onDone();
    } catch {
      toast.error("Gagal menyimpan paket. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label>Kode Paket</Label>
        <Input
          value={form.code}
          onChange={(e) => setField("code", e.target.value)}
          placeholder="mis. pro"
          disabled={!!editing}
          aria-invalid={!!errors.code}
        />
        {editing && <p className="text-xs text-muted">Kode tidak dapat diubah setelah dibuat.</p>}
        <FieldError msg={errors.code} />
      </div>
      <div className="grid gap-2">
        <Label>Nama Paket</Label>
        <Input
          value={form.name}
          onChange={(e) => setField("name", e.target.value)}
          placeholder="mis. Paket Pro"
          aria-invalid={!!errors.name}
        />
        <FieldError msg={errors.name} />
      </div>
      <div className="grid gap-2">
        <Label>Harga</Label>
        <MoneyInput
          value={form.price}
          onChange={(n) => setField("price", n)}
          placeholder="0"
          aria-invalid={!!errors.price}
        />
        <FieldError msg={errors.price} />
      </div>
      <div className="grid gap-2">
        <Label>Periode (hari)</Label>
        <Input
          type="number"
          value={form.periodDays}
          onChange={(e) => setField("periodDays", Number(e.target.value))}
          placeholder="30"
          aria-invalid={!!errors.periodDays}
        />
        <FieldError msg={errors.periodDays} />
      </div>
      <label className="flex items-center gap-2 text-sm text-text">
        <Checkbox
          checked={form.isActive}
          onChange={(e) => setField("isActive", e.target.checked)}
        />
        Aktif (tampil di pemilihan paket tenant)
      </label>
      <div className="flex justify-end">
        <Button loading={busy} onClick={submit}>
          {editing ? "Simpan Perubahan" : "Tambah Paket"}
        </Button>
      </div>
    </div>
  );
}
