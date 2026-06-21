import { useMemo, useState } from "react";
import { Plus, MoreHorizontal, Pencil, Trash2, KeyRound } from "lucide-react";
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
      toast.error(e instanceof Error ? e.message : "Gagal menghapus staf");
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
          <ErrorState message={`Gagal memuat staf. ${staffQuery.error}`} onRetry={refresh} />
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
                    <StaffRoleBadge role={c.role} />
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
          onDone={() => {
            setFormOpen(false);
            setEditing(null);
            refresh();
          }}
        />
      </Modal>

      {/* Reset password */}
      <ResetPasswordModal target={resetTarget} onClose={() => setResetTarget(null)} />

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

function StaffForm({ editing, onDone }: { editing: Staff | null; onDone: () => void }) {
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

  const submit = async () => {
    const email = form.email.trim() || undefined;
    setBusy(true);
    try {
      if (editing) {
        const body: StaffUpdateInput = {
          name: form.name.trim(),
          username: form.username.trim(),
          email,
          role: form.role,
          status: form.status,
        };
        const parsed = staffUpdateSchema.safeParse(body);
        if (!parsed.success) {
          toast.error(parsed.error.issues[0]?.message ?? "Periksa input.");
          return;
        }
        await staffService.update(editing.id, body);
        toast.success("Data staf diperbarui");
      } else {
        const body: StaffCreateInput = {
          name: form.name.trim(),
          username: form.username.trim(),
          email,
          password: form.password,
          role: form.role,
          status: form.status,
        };
        const parsed = staffCreateSchema.safeParse(body);
        if (!parsed.success) {
          toast.error(parsed.error.issues[0]?.message ?? "Periksa input.");
          return;
        }
        await staffService.create(body);
        toast.success("Staf ditambahkan");
      }
      onDone();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Gagal menyimpan staf");
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
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          placeholder="mis. Rini Wulandari"
        />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="grid gap-2">
          <Label>Username</Label>
          <Input
            value={form.username}
            onChange={(e) => setForm({ ...form, username: e.target.value })}
            placeholder="mis. rini"
          />
        </div>
        {!editing && (
          <div className="grid gap-2">
            <Label>Password</Label>
            <Input
              type="text"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              placeholder="mis. kasir123"
            />
          </div>
        )}
      </div>
      <div className="grid gap-2">
        <Label>Email</Label>
        <Input
          type="email"
          value={form.email}
          onChange={(e) => setForm({ ...form, email: e.target.value })}
          placeholder="rini@elkasir.id"
        />
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
      toast.error(e instanceof Error ? e.message : "Gagal mereset password");
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
