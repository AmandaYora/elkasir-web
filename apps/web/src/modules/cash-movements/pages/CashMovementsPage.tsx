import { useMemo, useState } from "react";
import { Plus, ArrowDownCircle, ArrowUpCircle, Scale } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
import { Select } from "@/shared/components/ui/select";
import { Textarea } from "@/shared/components/ui/textarea";
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
import { Modal } from "@/shared/components/ui/modal";
import { FieldError } from "@/shared/components/ui/field-error";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { zodFieldErrors } from "@/shared/lib/form";
import { formatIDR, formatDateTime } from "@/shared/lib/formatter";
import { useAsync } from "@/shared/hooks/useAsync";
import { cashMovementsService } from "@/modules/cash-movements/services/cash-movements.service";
import { cashMovementSchema } from "@/modules/cash-movements/schemas/cash-movement.schema";
import { CashMovementTypeBadge } from "@/modules/cash-movements/components/CashMovementTypeBadge";
import type {
  CashMovement,
  CashMovementInput,
  CashMovementType,
} from "@/modules/cash-movements/types/cash-movement.types";

// Nilai bertanda: capital selalu masuk (+); expense selalu keluar (-);
// adjustment mengikuti tanda amount.
const signedAmount = (m: CashMovement) => {
  if (m.type === "capital") return Math.abs(m.amount);
  if (m.type === "expense") return -Math.abs(m.amount);
  return m.amount;
};

export default function CashMovementsPage() {
  const movementsQuery = useAsync(() => cashMovementsService.list({ limit: 200 }), []);

  const items = movementsQuery.data?.data ?? [];

  const [open, setOpen] = useState(false);

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
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">Mutasi Kas</h2>
          <p className="text-sm text-muted">Pantau setiap kas masuk dan keluar dari laci.</p>
        </div>
        <Button size="sm" onClick={() => setOpen(true)}>
          <Plus className="h-4 w-4" /> Mutasi Baru
        </Button>
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

      <Modal
        open={open}
        onClose={() => setOpen(false)}
        title="Mutasi Kas Baru"
        description="Catat modal, biaya, atau penyesuaian kas pada laci."
      >
        <CashMovementForm
          key={open ? "open" : "closed"}
          onDone={() => {
            setOpen(false);
            refresh();
          }}
        />
      </Modal>
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

function CashMovementForm({ onDone }: { onDone: () => void }) {
  const [type, setType] = useState<CashMovementType>("capital");
  const [amount, setAmount] = useState("");
  const [notes, setNotes] = useState("");
  const [approvedBy, setApprovedBy] = useState("");
  const [busy, setBusy] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const clearError = (key: string) => setErrors((e) => (e[key] ? { ...e, [key]: "" } : e));

  const submit = async () => {
    const body: CashMovementInput = {
      type,
      amount: Number(amount),
      notes: notes.trim() || undefined,
      approvedBy: approvedBy.trim() || undefined,
    };
    const parsed = cashMovementSchema.safeParse(body);
    if (!parsed.success) {
      setErrors(zodFieldErrors(parsed.error));
      return;
    }
    setBusy(true);
    try {
      await cashMovementsService.create(body);
      toast.success("Mutasi kas dicatat");
      onDone();
    } catch (e) {
      toast.error("Gagal menyimpan mutasi. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label>Jenis</Label>
        <Select value={type} onChange={(e) => setType(e.target.value as CashMovementType)}>
          <option value="capital">Modal</option>
          <option value="expense">Biaya</option>
          <option value="adjustment">Penyesuaian</option>
        </Select>
      </div>
      <div className="grid gap-2">
        <Label>Nominal (IDR)</Label>
        {/* Number input (bukan MoneyInput): jenis "adjustment" boleh negatif untuk mengurangi
            kas — masker ribuan akan membuang tanda minus. */}
        <Input
          type="number"
          placeholder="0"
          value={amount}
          onChange={(e) => {
            setAmount(e.target.value);
            clearError("amount");
          }}
          aria-invalid={!!errors.amount}
        />
        {type === "adjustment" && (
          <p className="text-xs text-muted">Boleh negatif untuk mengurangi kas (mis. -5000).</p>
        )}
        <FieldError msg={errors.amount} />
      </div>
      <div className="grid gap-2">
        <Label>Catatan</Label>
        <Textarea
          placeholder="Alasan / keterangan"
          rows={3}
          value={notes}
          onChange={(e) => {
            setNotes(e.target.value);
            clearError("notes");
          }}
          aria-invalid={!!errors.notes}
        />
        <FieldError msg={errors.notes} />
      </div>
      <div className="grid gap-2">
        <Label>Disetujui oleh (opsional)</Label>
        <Input
          placeholder="Nama pemberi persetujuan"
          value={approvedBy}
          onChange={(e) => {
            setApprovedBy(e.target.value);
            clearError("approvedBy");
          }}
          aria-invalid={!!errors.approvedBy}
        />
        <FieldError msg={errors.approvedBy} />
      </div>
      <div className="flex justify-end">
        <Button loading={busy} onClick={submit}>
          Simpan Mutasi
        </Button>
      </div>
    </div>
  );
}
