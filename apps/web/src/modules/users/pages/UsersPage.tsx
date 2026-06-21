import { useMemo, useState } from "react";
import { Search, Plus, MoreHorizontal, Pencil, Trash2, KeyRound } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
import { Select } from "@/shared/components/ui/select";
import { Card, CardContent } from "@/shared/components/ui/card";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/shared/components/ui/table";
import { Dropdown, DropdownItem } from "@/shared/components/ui/dropdown";
import { Modal } from "@/shared/components/ui/modal";
import { ConfirmDialog } from "@/shared/components/ui/confirm-dialog";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { formatDateTime } from "@/shared/lib/formatter";
import { useAsync } from "@/shared/hooks/useAsync";
import { usersService } from "@/modules/users/services/users.service";
import { adminCreateSchema, adminUpdateSchema } from "@/modules/users/schemas/user.schema";
import { AdminRoleBadge, AdminStatusBadge } from "@/modules/users/components/AdminRoleBadge";
import type {
  ActiveStatus,
  AdminCreateInput,
  AdminRole,
  AdminUpdateInput,
  AdminUser,
} from "@/modules/users/types/user.types";

interface FormState {
  name: string;
  email: string;
  password: string;
  role: AdminRole;
  status: ActiveStatus;
}

const emptyForm: FormState = {
  name: "",
  email: "",
  password: "",
  role: "manager",
  status: "active",
};

export default function UsersPage() {
  const usersQuery = useAsync(() => usersService.list({ limit: 200 }), []);
  const list = usersQuery.data?.data ?? [];

  const [q, setQ] = useState("");
  const [roleFilter, setRoleFilter] = useState("all");
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<AdminUser | null>(null);
  const [deleting, setDeleting] = useState<AdminUser | null>(null);
  const [resetTarget, setResetTarget] = useState<AdminUser | null>(null);

  const refresh = () => usersQuery.refetch();

  const filtered = useMemo(
    () =>
      list.filter(
        (u) =>
          (roleFilter === "all" || u.role === roleFilter) &&
          (q === "" ||
            u.name.toLowerCase().includes(q.toLowerCase()) ||
            u.email.toLowerCase().includes(q.toLowerCase())),
      ),
    [list, q, roleFilter],
  );

  const [busy, setBusy] = useState(false);
  const remove = async () => {
    if (!deleting) return;
    setBusy(true);
    try {
      await usersService.remove(deleting.id);
      toast.success("Pengguna berhasil dihapus");
      setDeleting(null);
      refresh();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Gagal menghapus pengguna");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">Pengguna Admin</h2>
          <p className="text-sm text-muted">
            Kelola siapa yang dapat mengakses dashboard admin web (berbeda dari Staf POS).
          </p>
        </div>
        <Button
          size="sm"
          onClick={() => {
            setEditing(null);
            setFormOpen(true);
          }}
        >
          <Plus className="h-4 w-4" /> Tambah Pengguna
        </Button>
      </div>

      <Card>
        <CardContent className="p-4">
          <div className="flex flex-wrap items-center gap-2">
            <div className="relative min-w-[240px] flex-1">
              <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted" />
              <Input
                value={q}
                onChange={(e) => setQ(e.target.value)}
                placeholder="Cari nama atau email…"
                className="pl-9"
              />
            </div>
            <Select
              value={roleFilter}
              onChange={(e) => setRoleFilter(e.target.value)}
              className="w-[160px]"
            >
              <option value="all">Semua peran</option>
              <option value="owner">Pemilik</option>
              <option value="admin">Admin</option>
              <option value="manager">Manajer</option>
              <option value="viewer">Viewer</option>
            </Select>
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        {usersQuery.loading ? (
          <LoadingState label="Memuat pengguna…" />
        ) : usersQuery.error ? (
          <ErrorState message={`Gagal memuat pengguna. ${usersQuery.error}`} onRetry={refresh} />
        ) : filtered.length === 0 ? (
          <EmptyState title="Tidak ada pengguna yang cocok" />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Pengguna</TableHead>
                <TableHead>Peran</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Aktivitas terakhir</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((u) => (
                <TableRow key={u.id}>
                  <TableCell>
                    <div className="font-medium">{u.name}</div>
                    <div className="text-xs text-muted">{u.email}</div>
                  </TableCell>
                  <TableCell>
                    <AdminRoleBadge role={u.role} />
                  </TableCell>
                  <TableCell>
                    <AdminStatusBadge status={u.status} />
                  </TableCell>
                  <TableCell className="text-sm text-muted">
                    {u.lastActiveAt ? formatDateTime(u.lastActiveAt) : "—"}
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
                          setEditing(u);
                          setFormOpen(true);
                        }}
                      >
                        <Pencil className="h-3.5 w-3.5" /> Ubah
                      </DropdownItem>
                      <DropdownItem onClick={() => setResetTarget(u)}>
                        <KeyRound className="h-3.5 w-3.5" /> Reset password
                      </DropdownItem>
                      <DropdownItem danger onClick={() => setDeleting(u)}>
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
        title={editing ? "Ubah Pengguna" : "Tambah Pengguna"}
        description="Email & password dipakai untuk login ke dashboard admin web."
      >
        <UserForm
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
        title="Hapus pengguna ini?"
        description={deleting ? `Tindakan ini mencabut akses dashboard untuk "${deleting.name}".` : ""}
        confirmLabel="Hapus"
        danger
        loading={busy}
        onConfirm={remove}
        onClose={() => setDeleting(null)}
      />
    </div>
  );
}

function UserForm({ editing, onDone }: { editing: AdminUser | null; onDone: () => void }) {
  const [form, setForm] = useState<FormState>(
    editing
      ? {
          name: editing.name,
          email: editing.email,
          password: "",
          role: editing.role,
          status: editing.status,
        }
      : emptyForm,
  );
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    setBusy(true);
    try {
      if (editing) {
        const body: AdminUpdateInput = {
          name: form.name.trim(),
          email: form.email.trim(),
          role: form.role,
          status: form.status,
        };
        const parsed = adminUpdateSchema.safeParse(body);
        if (!parsed.success) {
          toast.error(parsed.error.issues[0]?.message ?? "Periksa input.");
          return;
        }
        await usersService.update(editing.id, body);
        toast.success("Pengguna berhasil diperbarui");
      } else {
        const body: AdminCreateInput = {
          name: form.name.trim(),
          email: form.email.trim(),
          password: form.password,
          role: form.role,
          status: form.status,
        };
        const parsed = adminCreateSchema.safeParse(body);
        if (!parsed.success) {
          toast.error(parsed.error.issues[0]?.message ?? "Periksa input.");
          return;
        }
        await usersService.create(body);
        toast.success("Pengguna berhasil ditambahkan");
      }
      onDone();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Gagal menyimpan pengguna");
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
          placeholder="mis. Sari Melati"
        />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="grid gap-2">
          <Label>Email</Label>
          <Input
            type="email"
            value={form.email}
            onChange={(e) => setForm({ ...form, email: e.target.value })}
            placeholder="sari@elkasir.id"
          />
        </div>
        {!editing && (
          <div className="grid gap-2">
            <Label>Password</Label>
            <Input
              type="text"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              placeholder="mis. admin123"
            />
          </div>
        )}
      </div>
      <div className="grid gap-2">
        <Label>Peran</Label>
        <Select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value as AdminRole })}>
          <option value="owner">Pemilik</option>
          <option value="admin">Admin</option>
          <option value="manager">Manajer</option>
          <option value="viewer">Viewer</option>
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
      <div className="flex justify-end">
        <Button loading={busy} onClick={submit}>
          {editing ? "Simpan Perubahan" : "Tambah Pengguna"}
        </Button>
      </div>
    </div>
  );
}

function ResetPasswordModal({ target, onClose }: { target: AdminUser | null; onClose: () => void }) {
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
      await usersService.resetPassword(target.id, password);
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
      description={target ? `Tetapkan password baru untuk "${target.name}".` : undefined}
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
          placeholder="mis. admin123"
        />
      </div>
    </Modal>
  );
}
