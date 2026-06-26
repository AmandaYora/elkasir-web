import { useMemo, useState } from "react";
import { Plus, CheckCircle2, Clock } from "lucide-react";
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
import { MoneyInput } from "@/shared/components/ui/money-input";
import { FieldError } from "@/shared/components/ui/field-error";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { zodFieldErrors } from "@/shared/lib/form";
import { formatIDR, formatDateTime } from "@/shared/lib/formatter";
import { useAsync } from "@/shared/hooks/useAsync";
import { withdrawalsService } from "@/modules/withdrawals/services/withdrawals.service";
import { withdrawalSchema } from "@/modules/withdrawals/schemas/withdrawal.schema";
import { WithdrawalStatusBadge } from "@/modules/withdrawals/components/WithdrawalStatusBadge";
import type { Withdrawal, WithdrawalInput } from "@/modules/withdrawals/types/withdrawal.types";

export default function WithdrawalsPage() {
  const withdrawalsQuery = useAsync(() => withdrawalsService.list({ limit: 200 }), []);

  const items = withdrawalsQuery.data?.data ?? [];

  const [detail, setDetail] = useState<Withdrawal | null>(null);
  const [createOpen, setCreateOpen] = useState(false);

  const totalSuccess = useMemo(
    () => items.filter((w) => w.status === "success").reduce((a, w) => a + w.amount, 0),
    [items],
  );
  const pending = useMemo(
    () => items.filter((w) => w.status === "pending" || w.status === "processing").length,
    [items],
  );

  const refresh = () => withdrawalsQuery.refetch();

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">Penarikan Dana</h2>
          <p className="text-sm text-muted">Tarik dana hasil penjualan ke rekening bank Anda.</p>
        </div>
        <Button size="sm" onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" /> Ajukan Penarikan
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <StatCard
          label="Pengajuan Tertunda"
          value={String(pending)}
          hint="dalam antrean"
          icon={Clock}
          tone="text-warning"
        />
        <StatCard
          label="Total Ditarik"
          value={formatIDR(totalSuccess)}
          hint="berhasil dicairkan"
          icon={CheckCircle2}
          tone="text-success"
        />
      </div>

      <Card className="overflow-hidden">
        {withdrawalsQuery.loading ? (
          <LoadingState />
        ) : withdrawalsQuery.error ? (
          <ErrorState message="Gagal memuat penarikan. Coba lagi." onRetry={refresh} />
        ) : items.length === 0 ? (
          <EmptyState title="Belum ada penarikan." description="Ajukan penarikan pertama Anda." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Referensi</TableHead>
                <TableHead>Diajukan</TableHead>
                <TableHead>Tujuan</TableHead>
                <TableHead>Pemilik Rekening</TableHead>
                <TableHead className="text-right">Jumlah</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((w) => (
                <TableRow key={w.id} className="cursor-pointer" onClick={() => setDetail(w)}>
                  <TableCell className="font-mono text-xs font-medium">
                    {w.reference ?? "—"}
                  </TableCell>
                  <TableCell className="text-sm text-muted">
                    {formatDateTime(w.createdAt)}
                  </TableCell>
                  <TableCell className="text-sm">
                    {w.bank} <span className="font-mono text-xs text-muted">{w.account}</span>
                  </TableCell>
                  <TableCell className="text-sm">{w.holder}</TableCell>
                  <TableCell className="text-right text-sm font-semibold">
                    {formatIDR(w.amount)}
                  </TableCell>
                  <TableCell>
                    <WithdrawalStatusBadge status={w.status} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {/* Detail drawer */}
      <Drawer
        open={!!detail}
        onClose={() => setDetail(null)}
        title={detail?.reference ?? detail?.id}
        description={detail ? formatDateTime(detail.createdAt) : undefined}
      >
        {detail && (
          <div className="space-y-4">
            <div className="rounded-lg border border-border bg-surface-muted p-5 text-center">
              <div className="text-[11px] uppercase tracking-wider text-muted">Jumlah</div>
              <div className="mt-1 text-3xl font-semibold tracking-tight text-text">
                {formatIDR(detail.amount)}
              </div>
              <div className="mt-2 flex justify-center">
                <WithdrawalStatusBadge status={detail.status} />
              </div>
            </div>
            <div className="divide-y divide-border rounded-lg border border-border">
              {[
                ["Bank", detail.bank],
                ["Rekening", detail.account],
                ["Pemilik Rekening", detail.holder],
                ["Referensi", detail.reference ?? "—"],
              ].map(([k, v]) => (
                <div key={k} className="flex items-center justify-between px-4 py-3 text-sm">
                  <span className="text-muted">{k}</span>
                  <span className="font-medium">{v}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </Drawer>

      {/* Create form */}
      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Penarikan Baru"
        description="Dana biasanya cair dalam 1-2 hari kerja."
      >
        <WithdrawalForm
          key={createOpen ? "open" : "closed"}
          onDone={() => {
            setCreateOpen(false);
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
  icon: typeof Clock;
  tone: string;
}) {
  return (
    <Card>
      <CardContent className="flex items-start justify-between p-4">
        <div>
          <div className="text-sm text-muted">{label}</div>
          <div className="mt-1 text-2xl font-semibold text-text">{value}</div>
          <div className="mt-1 text-xs text-muted">{hint}</div>
        </div>
        <Icon className={`h-5 w-5 ${tone}`} />
      </CardContent>
    </Card>
  );
}

function WithdrawalForm({ onDone }: { onDone: () => void }) {
  const [form, setForm] = useState<WithdrawalInput>({
    amount: 0,
    bank: "",
    account: "",
    holder: "",
  });
  const [busy, setBusy] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const set = <K extends keyof WithdrawalInput>(key: K, value: WithdrawalInput[K]) => {
    setForm((f) => ({ ...f, [key]: value }));
    setErrors((e) => (e[key] ? { ...e, [key]: "" } : e));
  };

  const submit = async () => {
    const parsed = withdrawalSchema.safeParse(form);
    if (!parsed.success) {
      setErrors(zodFieldErrors(parsed.error));
      return;
    }
    setBusy(true);
    try {
      await withdrawalsService.create(form);
      toast.success("Penarikan berhasil diajukan");
      onDone();
    } catch (e) {
      toast.error("Gagal mengajukan penarikan. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label>Jumlah (IDR)</Label>
        <MoneyInput
          placeholder="0"
          value={form.amount}
          onChange={(n) => set("amount", n)}
          aria-invalid={!!errors.amount}
        />
        <FieldError msg={errors.amount} />
      </div>
      <div className="grid gap-2">
        <Label>Bank</Label>
        <Select
          value={form.bank}
          onChange={(e) => set("bank", e.target.value)}
          aria-invalid={!!errors.bank}
        >
          <option value="">Pilih bank</option>
          <option value="BCA">BCA</option>
          <option value="Mandiri">Mandiri</option>
          <option value="BNI">BNI</option>
          <option value="BRI">BRI</option>
        </Select>
        <FieldError msg={errors.bank} />
      </div>
      <div className="grid gap-2">
        <Label>Nomor Rekening</Label>
        <Input
          placeholder="**** ****"
          value={form.account}
          onChange={(e) => set("account", e.target.value)}
          aria-invalid={!!errors.account}
        />
        <FieldError msg={errors.account} />
      </div>
      <div className="grid gap-2">
        <Label>Pemilik Rekening</Label>
        <Input
          placeholder="Nama lengkap"
          value={form.holder}
          onChange={(e) => set("holder", e.target.value)}
          aria-invalid={!!errors.holder}
        />
        <FieldError msg={errors.holder} />
      </div>
      <div className="flex justify-end">
        <Button loading={busy} onClick={submit}>
          Ajukan Penarikan
        </Button>
      </div>
    </div>
  );
}
