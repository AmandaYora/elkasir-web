import { useState } from "react";
import { Plus, MoreHorizontal, KeyRound, Ban, CheckCircle2, Webhook, Copy } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
import { Card } from "@/shared/components/ui/card";
import { Badge } from "@/shared/components/ui/badge";
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
import { ConfirmDialog } from "@/shared/components/ui/confirm-dialog";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { useAsync } from "@/shared/hooks/useAsync";
import { ApiError } from "@/shared/types/api";
import { platformService } from "@/modules/platform/services/platform.service";
import type {
  PaymentApp,
  CreatePaymentAppInput,
  CreatePaymentAppResult,
} from "@/modules/platform/types/platform.types";

// Aplikasi Terdaftar (PLAN.md §9.1.3/§9.1.11/§9.3 PF1) — registry APP-ID+SECRET yang boleh
// membuat tagihan lewat SATU dompet gateway yang sama (§9.1.1). Dua baris "internal" (bawaan
// sistem — self-order & langganan tenant) tidak bisa dihapus/dinonaktifkan lewat halaman ini
// (mereka menopang trafik produksi langsung). Aplikasi "external" boleh didaftarkan sekarang
// meski API eksternalnya sendiri (§9.7) belum tersedia — makanya selalu diberi label jelas.
export default function PlatformPaymentClientsPage() {
  const appsQuery = useAsync(() => platformService.listPaymentApps(), []);
  const list = appsQuery.data ?? [];

  const [formOpen, setFormOpen] = useState(false);
  const [revealed, setRevealed] = useState<CreatePaymentAppResult | null>(null);
  const [resetting, setResetting] = useState<PaymentApp | null>(null);
  const [resetSecret, setResetSecret] = useState<string | null>(null);
  const [toggling, setToggling] = useState<PaymentApp | null>(null);
  const [busy, setBusy] = useState(false);

  const refresh = () => appsQuery.refetch();

  const doReset = async () => {
    if (!resetting) return;
    setBusy(true);
    try {
      const { secret } = await platformService.resetPaymentAppSecret(resetting.id);
      setResetSecret(secret);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "Gagal mereset secret. Coba lagi.");
      setResetting(null);
    } finally {
      setBusy(false);
    }
  };

  const toggleStatus = async () => {
    if (!toggling) return;
    const nextStatus = toggling.status === "active" ? "inactive" : "active";
    setBusy(true);
    try {
      await platformService.setPaymentAppStatus(toggling.id, nextStatus);
      toast.success(
        nextStatus === "inactive"
          ? "Aplikasi berhasil dinonaktifkan"
          : "Aplikasi berhasil diaktifkan",
      );
      setToggling(null);
      refresh();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "Gagal mengubah status aplikasi. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">Aplikasi Terdaftar</h2>
          <p className="text-sm text-muted">
            {list.length} aplikasi terdaftar untuk gateway pembayaran bersama.
          </p>
        </div>
        <Button size="sm" onClick={() => setFormOpen(true)}>
          <Plus className="h-4 w-4" /> Daftarkan Aplikasi
        </Button>
      </div>

      <Card className="overflow-hidden">
        {appsQuery.loading ? (
          <LoadingState />
        ) : appsQuery.error ? (
          <ErrorState message="Gagal memuat aplikasi. Coba lagi." onRetry={refresh} />
        ) : list.length === 0 ? (
          <EmptyState title="Belum ada aplikasi" description="Daftarkan aplikasi pertama." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nama Aplikasi</TableHead>
                <TableHead>APP-ID</TableHead>
                <TableHead>Jenis</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Dibuat</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((a) => {
                const isInternal = a.kind === "internal";
                return (
                  <TableRow key={a.id}>
                    <TableCell>
                      <div className="flex items-center gap-2.5">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                          <Webhook className="h-4 w-4" />
                        </div>
                        <span className="text-sm font-medium">{a.name}</span>
                      </div>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted">{a.appId}</TableCell>
                    <TableCell>
                      <div className="flex flex-col gap-1">
                        <Badge tone={isInternal ? "primary" : "neutral"}>
                          {isInternal ? "Internal" : "Eksternal"}
                        </Badge>
                        {!isInternal && (
                          <span className="text-[11px] text-muted">
                            API eksternal belum tersedia
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge tone={a.status === "active" ? "success" : "neutral"}>
                        {a.status === "active" ? "Aktif" : "Nonaktif"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted">
                      {new Date(a.createdAt).toLocaleDateString("id-ID")}
                    </TableCell>
                    <TableCell>
                      <Dropdown
                        trigger={
                          <Button variant="ghost" size="icon" disabled={isInternal}>
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        }
                      >
                        <DropdownItem onClick={() => setResetting(a)}>
                          <KeyRound className="h-3.5 w-3.5" /> Reset Secret
                        </DropdownItem>
                        <DropdownItem danger={a.status === "active"} onClick={() => setToggling(a)}>
                          {a.status === "active" ? (
                            <>
                              <Ban className="h-3.5 w-3.5" /> Nonaktifkan
                            </>
                          ) : (
                            <>
                              <CheckCircle2 className="h-3.5 w-3.5" /> Aktifkan
                            </>
                          )}
                        </DropdownItem>
                      </Dropdown>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </Card>

      <Modal open={formOpen} onClose={() => setFormOpen(false)} title="Daftarkan Aplikasi Baru">
        <CreateAppForm
          onDone={(result) => {
            setFormOpen(false);
            setRevealed(result);
            refresh();
          }}
        />
      </Modal>

      <SecretRevealModal
        title="Aplikasi Terdaftar"
        appId={revealed?.appId}
        secret={revealed?.secret ?? null}
        onClose={() => setRevealed(null)}
      />
      <SecretRevealModal
        title="Secret Direset"
        appId={resetting?.appId}
        secret={resetSecret}
        onClose={() => {
          setResetting(null);
          setResetSecret(null);
          refresh();
        }}
      />

      <ConfirmDialog
        open={!!resetting && !resetSecret}
        title="Reset secret aplikasi ini?"
        description={
          resetting
            ? `Secret lama "${resetting.name}" akan langsung tidak berlaku. Pastikan aplikasi pemanggil siap menerima secret baru.`
            : ""
        }
        confirmLabel="Reset Secret"
        danger
        loading={busy}
        onConfirm={doReset}
        onClose={() => setResetting(null)}
      />

      <ConfirmDialog
        open={!!toggling}
        title={
          toggling?.status === "active" ? "Nonaktifkan aplikasi ini?" : "Aktifkan aplikasi ini?"
        }
        description={
          toggling
            ? `"${toggling.name}" ${toggling.status === "active" ? "tidak akan bisa" : "akan bisa kembali"} membuat tagihan baru.`
            : ""
        }
        confirmLabel={toggling?.status === "active" ? "Nonaktifkan" : "Aktifkan"}
        danger={toggling?.status === "active"}
        loading={busy}
        onConfirm={toggleStatus}
        onClose={() => setToggling(null)}
      />
    </div>
  );
}

function CreateAppForm({ onDone }: { onDone: (result: CreatePaymentAppResult) => void }) {
  const [form, setForm] = useState<CreatePaymentAppInput>({ name: "", callbackUrl: "" });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!form.name.trim()) {
      setError("Nama aplikasi wajib diisi.");
      return;
    }
    setBusy(true);
    try {
      const result = await platformService.createPaymentApp(form);
      toast.success("Aplikasi berhasil didaftarkan");
      onDone(result);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "Gagal mendaftarkan aplikasi. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label>Nama Aplikasi</Label>
        <Input
          value={form.name}
          onChange={(e) => {
            setForm((f) => ({ ...f, name: e.target.value }));
            if (error) setError("");
          }}
          placeholder="mis. Toko Sebelah"
          aria-invalid={!!error}
        />
        <FieldError msg={error} />
      </div>
      <div className="grid gap-2">
        <Label>Callback URL (opsional)</Label>
        <Input
          value={form.callbackUrl}
          onChange={(e) => setForm((f) => ({ ...f, callbackUrl: e.target.value }))}
          placeholder="https://aplikasi-anda.com/webhook"
        />
      </div>
      <p className="text-xs text-muted">
        API eksternal belum tersedia — mendaftarkan aplikasi sekarang menyiapkan APP-ID + SECRET
        lebih awal, tapi belum bisa dipakai memanggil dari luar.
      </p>
      <div className="flex justify-end">
        <Button loading={busy} onClick={submit}>
          Daftarkan
        </Button>
      </div>
    </div>
  );
}

// Menampilkan secret SEKALI — dipakai baik untuk pembuatan aplikasi baru maupun reset secret
// (§9.1.3). Tidak pernah ditampilkan lagi setelah modal ini ditutup.
function SecretRevealModal({
  title,
  appId,
  secret,
  onClose,
}: {
  title: string;
  appId?: string;
  secret: string | null;
  onClose: () => void;
}) {
  const copy = () => {
    if (!secret) return;
    navigator.clipboard.writeText(secret).then(() => toast.success("Secret disalin"));
  };

  return (
    <Modal
      open={!!secret}
      onClose={onClose}
      title={title}
      description={appId ? `APP-ID: ${appId}` : undefined}
    >
      <div className="grid gap-4">
        <div className="rounded-xl border border-warning bg-warning-soft p-3 text-xs text-warning">
          Simpan secret ini sekarang — tidak akan ditampilkan lagi setelah jendela ini ditutup.
        </div>
        <div className="flex items-center gap-2">
          <code className="flex-1 truncate rounded-lg border border-border bg-surface-muted px-3 py-2 font-mono text-xs">
            {secret}
          </code>
          <Button variant="outline" size="icon" onClick={copy} type="button">
            <Copy className="h-4 w-4" />
          </Button>
        </div>
        <div className="flex justify-end">
          <Button onClick={onClose}>Sudah disimpan</Button>
        </div>
      </div>
    </Modal>
  );
}
