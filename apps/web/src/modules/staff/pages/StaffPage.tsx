import { useMemo, useState } from "react";
import { Plus, MoreHorizontal, Pencil, Trash2, KeyRound, Hash } from "lucide-react";
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
import { Dropdown, DropdownItem } from "@/shared/components/ui/dropdown";
import { Modal } from "@/shared/components/ui/modal";
import { ConfirmDialog } from "@/shared/components/ui/confirm-dialog";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { useAsync } from "@/shared/hooks/useAsync";
import { staffService } from "@/modules/staff/services/staff.service";
import { staffCreateSchema, staffUpdateSchema } from "@/modules/staff/schemas/staff.schema";
import { zodFieldErrors } from "@/shared/lib/form";
import { FieldError } from "@/shared/components/ui/field-error";
import { StaffRoleBadge, StaffStatusBadge } from "@/modules/staff/components/StaffRoleBadge";
import type {
  ActiveStatus,
  Staff,
  StaffCreateInput,
  StaffRole,
  StaffUpdateInput,
} from "@/modules/staff/types/staff.types";

interface FormState {
  name: string;
  email: string;
  username: string;
  password: string;
  role: StaffRole;
  status: ActiveStatus;
}

const emptyForm: FormState = {
  name: "",
  email: "",
  username: "",
  password: "",
  role: "cashier",
  status: "active",
};

export default function StaffPage() {
  const staffQuery = useAsync(() => staffService.list({ limit: 200 }), []);
  const list = staffQuery.data?.data ?? [];

  const [statusFilter, setStatusFilter] = useState("all");
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Staff | null>(null);
  const [deleting, setDeleting] = useState<Staff | null>(null);
  const [resetTarget, setResetTarget] = useState<Staff | null>(null);
  const [pinTarget, setPinTarget] = useState<Staff | null>(null);

  const refresh = () => staffQuery.refetch();

  const data = useMemo(
    () => list.filter((c) => statusFilter === "all" || c.status === statusFilter),
    [list, statusFilter],
  );

  const [busy, setBusy] = useState(false);
  const remove = async () => {
    if (!deleting) return;
    setBusy(true);
    try {
      await staffService.remove(deleting.id);
      toast.success("Staf dihapus");
      setDeleting(null);
      refresh();
    } catch (e) {
      toast.error("Gagal menghapus staf. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">Staf</h2>
          <p className="text-sm text-muted">{list.length} anggota tim</p>
        </div>
        <Button
          size="sm"
          onClick={() => {
            setEditing(null);
            setFormOpen(true);
          }}
        >
          <Plus className="h-4 w-4" /> Tambah Staf
        </Button>
      </div>

      <Card>
        <CardContent className="p-4">
          <div className="flex flex-wrap items-center gap-2">
            <Select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="w-[160px]"
            >
              <option value="all">Semua status</option>
              <option value="active">Aktif</option>
              <option value="inactive">Nonaktif</option>
            </Select>
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        {staffQuery.loading ? (
          <LoadingState label="Memuat staf…" />
        ) : staffQuery.error ? (
          <ErrorState message="Gagal memuat staf. Coba lagi." onRetry={refresh} />
        ) : data.length === 0 ? (
          <EmptyState title="Belum ada staf" description="Tambahkan anggota tim pertama Anda." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Staf</TableHead>
                <TableHead>Username</TableHead>
                <TableHead>Peran</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((c) => (
                <TableRow key={c.id}>
                  <TableCell>
                    <div className="font-medium">{c.name}</div>
                    <div className="text-xs text-muted">{c.email || "—"}</div>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted">{c.username}</TableCell>
                  <TableCell>
                    <div className="flex flex-col items-start gap-1">
                      <StaffRoleBadge role={c.role} />
                      {c.role === "supervisor" &&
                        (c.hasPin ? (
                          <span className="text-[11px] font-medium text-muted">PIN aktif</span>
                        ) : (
                          <button
                            onClick={() => setPinTarget(c)}
                            className="text-[11px] font-semibold text-warning underline-offset-2 hover:underline"
                          >
                            PIN belum diatur — atur
                          </button>
                        ))}
                    </div>
                  </TableCell>
                  <TableCell>
                    <StaffStatusBadge status={c.status} />
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
                          setEditing(c);
                          setFormOpen(true);
                        }}
                      >
                        <Pencil className="h-3.5 w-3.5" /> Ubah
                      </DropdownItem>
                      <DropdownItem onClick={() => setResetTarget(c)}>
                        <KeyRound className="h-3.5 w-3.5" /> Reset password
                      </DropdownItem>
                      {c.role === "supervisor" && (
                        <DropdownItem onClick={() => setPinTarget(c)}>
                          <Hash className="h-3.5 w-3.5" /> Atur PIN supervisor
                        </DropdownItem>
                      )}
                      <DropdownItem danger onClick={() => setDeleting(c)}>
                        <Trash2 className="h-3.5 w-3.5" /> Hapus
                      </DropdownItem>
                    </Dropdown>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {/* Create / edit form */}
      <Modal
        open={formOpen}
        onClose={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        title={editing ? "Ubah Staf" : "Tambah Staf"}
        description="Username & password dipakai staf untuk login di aplikasi POS tablet."
      >
        <StaffForm
          key={editing?.id ?? "new"}
          editing={editing}
          onDone={(created) => {
            setFormOpen(false);
            setEditing(null);
            refresh();
            // Supervisor baru wajib punya PIN agar override di POS berfungsi → langsung tawarkan.
            if (created && created.role === "supervisor") setPinTarget(created);
          }}
        />
      </Modal>

      {/* Reset password */}
      <ResetPasswordModal target={resetTarget} onClose={() => setResetTarget(null)} />

      {/* Atur PIN supervisor */}
      <SetPinModal target={pinTarget} onClose={() => setPinTarget(null)} />

      <ConfirmDialog
        open={!!deleting}
        title="Hapus staf ini?"
        description={deleting ? `Tindakan ini menghapus "${deleting.name}" secara permanen.` : ""}
        confirmLabel="Hapus"
        danger
        loading={busy}
        onConfirm={remove}
        onClose={() => setDeleting(null)}
      />
    </div>
  );
}

function StaffForm({
  editing,
  onDone,
}: {
  editing: Staff | null;
  onDone: (created?: Staff) => void;
}) {
  const [form, setForm] = useState<FormState>(
    editing
      ? {
          name: editing.name,
          email: editing.email ?? "",
          username: editing.username,
          password: "",
          role: editing.role,
          status: editing.status,
        }
      : emptyForm,
  );
  const [busy, setBusy] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const setField = (key: keyof FormState, value: string) => {
    setForm((f) => ({ ...f, [key]: value }));
    setErrors((e) => (e[key] ? { ...e, [key]: "" } : e));
  };

  const submit = async () => {
    const email = form.email.trim() || undefined;
    const base = {
      name: form.name.trim(),
      username: form.username.trim(),
      email,
      role: form.role,
      status: form.status,
    };
    const parsed = editing
      ? staffUpdateSchema.safeParse(base)
      : staffCreateSchema.safeParse({ ...base, password: form.password });
    if (!parsed.success) {
      setErrors(zodFieldErrors(parsed.error));
      return;
    }
    setBusy(true);
    try {
      let created: Staff | undefined;
      if (editing) {
        await staffService.update(editing.id, base as StaffUpdateInput);
        toast.success("Data staf diperbarui");
      } else {
        created = await staffService.create({
          ...base,
          password: form.password,
        } as StaffCreateInput);
        toast.success("Staf ditambahkan");
      }
      onDone(created);
    } catch (e) {
      toast.error("Gagal menyimpan staf. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label>Nama Lengkap</Label>
        <Input
          value={form.name}
          onChange={(e) => setField("name", e.target.value)}
          placeholder="mis. Rini Wulandari"
          aria-invalid={!!errors.name}
        />
        <FieldError msg={errors.name} />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="grid gap-2">
          <Label>Username</Label>
          <Input
            value={form.username}
            onChange={(e) => setField("username", e.target.value)}
            placeholder="mis. rini"
            aria-invalid={!!errors.username}
          />
          <FieldError msg={errors.username} />
        </div>
        {!editing && (
          <div className="grid gap-2">
            <Label>Password</Label>
            <Input
              type="text"
              value={form.password}
              onChange={(e) => setField("password", e.target.value)}
              placeholder="mis. kasir123"
              aria-invalid={!!errors.password}
            />
            <FieldError msg={errors.password} />
          </div>
        )}
      </div>
      <div className="grid gap-2">
        <Label>Email</Label>
        <Input
          type="email"
          value={form.email}
          onChange={(e) => setField("email", e.target.value)}
          placeholder="rini@elkasir.id"
          aria-invalid={!!errors.email}
        />
        <FieldError msg={errors.email} />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="grid gap-2">
          <Label>Peran</Label>
          <Select
            value={form.role}
            onChange={(e) => setForm({ ...form, role: e.target.value as StaffRole })}
          >
            <option value="cashier">Kasir</option>
            <option value="supervisor">Supervisor</option>
          </Select>
        </div>
        <div className="grid gap-2">
          <Label>Status</Label>
          <Select
            value={form.status}
            onChange={(e) => setForm({ ...form, status: e.target.value as ActiveStatus })}
          >
            <option value="active">Aktif</option>
            <option value="inactive">Nonaktif</option>
          </Select>
        </div>
      </div>
      <div className="flex justify-end">
        <Button loading={busy} onClick={submit}>
          {editing ? "Simpan Perubahan" : "Tambah Staf"}
        </Button>
      </div>
    </div>
  );
}

function ResetPasswordModal({ target, onClose }: { target: Staff | null; onClose: () => void }) {
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!target) return;
    if (!password.trim()) {
      toast.error("Password baru wajib diisi");
      return;
    }
    setBusy(true);
    try {
      await staffService.resetPassword(target.id, password);
      toast.success("Password berhasil direset");
      setPassword("");
      onClose();
    } catch (e) {
      toast.error("Gagal mereset password. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      open={!!target}
      onClose={() => {
        setPassword("");
        onClose();
      }}
      title="Reset password"
      description={
        target
          ? `Tetapkan password baru untuk ${target.name}. Staf memakainya untuk login berikutnya.`
          : undefined
      }
      footer={
        <Button loading={busy} onClick={submit}>
          Simpan Password
        </Button>
      }
    >
      <div className="grid gap-2">
        <Label>Password baru</Label>
        <Input
          type="text"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="mis. kasir123"
        />
      </div>
    </Modal>
  );
}

// SetPinModal mengatur PIN persetujuan (approve-in-place) untuk supervisor. PIN 4–6 digit
// dipakai supervisor untuk mengotorisasi aksi kasir (diskon/selisih kas) langsung di POS.
function SetPinModal({ target, onClose }: { target: Staff | null; onClose: () => void }) {
  const [pin, setPin] = useState("");
  const [busy, setBusy] = useState(false);

  const close = () => {
    setPin("");
    onClose();
  };

  const submit = async (clear = false) => {
    if (!target) return;
    const value = clear ? "" : pin.trim();
    if (!clear && !/^\d{4,6}$/.test(value)) {
      toast.error("PIN harus 4–6 digit angka.");
      return;
    }
    setBusy(true);
    try {
      await staffService.setPin(target.id, value);
      toast.success(clear ? "PIN dihapus" : "PIN supervisor disimpan");
      close();
    } catch (e) {
      toast.error("Gagal menyimpan PIN. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      open={!!target}
      onClose={close}
      title="PIN supervisor"
      description={
        target
          ? `PIN 4–6 digit untuk ${target.name}. Dipakai menyetujui diskon/selisih kas di atas batas langsung di POS.`
          : undefined
      }
      footer={
        <div className="flex justify-between gap-2">
          <Button variant="ghost" onClick={() => submit(true)} disabled={busy}>
            Hapus PIN
          </Button>
          <Button loading={busy} onClick={() => submit(false)}>
            Simpan PIN
          </Button>
        </div>
      }
    >
      <div className="grid gap-2">
        <Label>PIN baru</Label>
        <Input
          inputMode="numeric"
          maxLength={6}
          value={pin}
          onChange={(e) => setPin(e.target.value.replace(/\D/g, ""))}
          placeholder="mis. 1234"
        />
      </div>
    </Modal>
  );
}
