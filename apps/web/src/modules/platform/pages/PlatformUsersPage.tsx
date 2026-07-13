import { useState } from "react";
import { Plus, MoreHorizontal, KeyRound, Ban, CheckCircle2, UserRound } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
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
import { ConfirmDialog } from "@/shared/components/ui/confirm-dialog";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { useAsync } from "@/shared/hooks/useAsync";
import { zodFieldErrors } from "@/shared/lib/form";
import { ApiError } from "@/shared/types/api";
import { platformService } from "@/modules/platform/services/platform.service";
import { usePlatformAuthStore } from "@/modules/platform/stores/platform-auth.store";
import {
  createPlatformUserSchema,
  resetPlatformUserPasswordSchema,
} from "@/modules/platform/schemas/platform-user.schema";
import { PlatformUserStatusBadge } from "@/modules/platform/components/PlatformUserStatusBadge";
import type {
  PlatformUser,
  CreatePlatformUserInput,
} from "@/modules/platform/types/platform.types";

// User Platform — superadmin account management (§2.9). No role tiers, never hard-delete
// (deactivate only), and a superadmin cannot deactivate their own account — the self-row's
// toggle is disabled client-side too, mirroring the backend guard.
export default function PlatformUsersPage() {
  const me = usePlatformAuthStore((s) => s.user);
  const usersQuery = useAsync(() => platformService.listUsers(), []);
  const list = usersQuery.data ?? [];

  const [formOpen, setFormOpen] = useState(false);
  const [resetting, setResetting] = useState<PlatformUser | null>(null);
  const [toggling, setToggling] = useState<PlatformUser | null>(null);
  const [busy, setBusy] = useState(false);

  const refresh = () => usersQuery.refetch();

  const toggleStatus = async () => {
    if (!toggling) return;
    const nextStatus = toggling.status === "active" ? "inactive" : "active";
    setBusy(true);
    try {
      await platformService.setUserStatus(toggling.id, nextStatus);
      toast.success(
        nextStatus === "inactive" ? "User berhasil dinonaktifkan" : "User berhasil diaktifkan",
      );
      setToggling(null);
      refresh();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "Gagal mengubah status user. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">User Platform</h2>
          <p className="text-sm text-muted">{list.length} akun superadmin</p>
        </div>
        <Button size="sm" onClick={() => setFormOpen(true)}>
          <Plus className="h-4 w-4" /> Tambah User
        </Button>
      </div>

      <Card className="overflow-hidden">
        {usersQuery.loading ? (
          <LoadingState />
        ) : usersQuery.error ? (
          <ErrorState message="Gagal memuat user. Coba lagi." onRetry={refresh} />
        ) : list.length === 0 ? (
          <EmptyState title="Belum ada user" description="Tambahkan akun superadmin pertama." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nama</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Dibuat</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((u) => {
                const isSelf = me?.id === u.id;
                return (
                  <TableRow key={u.id}>
                    <TableCell>
                      <div className="flex items-center gap-2.5">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                          <UserRound className="h-4 w-4" />
                        </div>
                        <span className="text-sm font-medium">
                          {u.name}
                          {isSelf && <span className="ml-1.5 text-xs text-muted">(Anda)</span>}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm text-muted">{u.email}</TableCell>
                    <TableCell>
                      <PlatformUserStatusBadge status={u.status} />
                    </TableCell>
                    <TableCell className="text-sm text-muted">
                      {new Date(u.createdAt).toLocaleDateString("id-ID")}
                    </TableCell>
                    <TableCell>
                      <Dropdown
                        trigger={
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        }
                      >
                        <DropdownItem onClick={() => setResetting(u)}>
                          <KeyRound className="h-3.5 w-3.5" /> Reset Password
                        </DropdownItem>
                        <DropdownItem
                          danger={u.status === "active"}
                          disabled={isSelf && u.status === "active"}
                          onClick={() => setToggling(u)}
                        >
                          {u.status === "active" ? (
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

      <Modal open={formOpen} onClose={() => setFormOpen(false)} title="Tambah User Platform">
        <CreateUserForm
          onDone={() => {
            setFormOpen(false);
            refresh();
          }}
        />
      </Modal>

      <ResetPasswordModal user={resetting} onClose={() => setResetting(null)} />

      <ConfirmDialog
        open={!!toggling}
        title={toggling?.status === "active" ? "Nonaktifkan user ini?" : "Aktifkan user ini?"}
        description={
          toggling
            ? `"${toggling.name}" ${toggling.status === "active" ? "tidak akan bisa" : "akan bisa kembali"} masuk ke Konsol Platform.`
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

function CreateUserForm({ onDone }: { onDone: () => void }) {
  const [form, setForm] = useState<CreatePlatformUserInput>({ name: "", email: "", password: "" });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

  const setField = <K extends keyof CreatePlatformUserInput>(
    key: K,
    value: CreatePlatformUserInput[K],
  ) => {
    setForm((f) => ({ ...f, [key]: value }));
    setErrors((e) => (e[key] ? { ...e, [key]: "" } : e));
  };

  const submit = async () => {
    const parsed = createPlatformUserSchema.safeParse(form);
    if (!parsed.success) {
      setErrors(zodFieldErrors(parsed.error));
      return;
    }
    setBusy(true);
    try {
      await platformService.createUser(parsed.data);
      toast.success("User platform berhasil dibuat");
      onDone();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "Gagal membuat user. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label>Nama</Label>
        <Input
          value={form.name}
          onChange={(e) => setField("name", e.target.value)}
          placeholder="mis. Dimas Prasetio"
          aria-invalid={!!errors.name}
        />
        <FieldError msg={errors.name} />
      </div>
      <div className="grid gap-2">
        <Label>Email</Label>
        <Input
          type="email"
          value={form.email}
          onChange={(e) => setField("email", e.target.value)}
          placeholder="nama@elkasir.app"
          aria-invalid={!!errors.email}
        />
        <FieldError msg={errors.email} />
      </div>
      <div className="grid gap-2">
        <Label>Password</Label>
        <Input
          type="password"
          value={form.password}
          onChange={(e) => setField("password", e.target.value)}
          placeholder="••••••••"
          aria-invalid={!!errors.password}
        />
        <FieldError msg={errors.password} />
      </div>
      <div className="flex justify-end">
        <Button loading={busy} onClick={submit}>
          Tambah User
        </Button>
      </div>
    </div>
  );
}

function ResetPasswordModal({ user, onClose }: { user: PlatformUser | null; onClose: () => void }) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!user) return;
    const parsed = resetPlatformUserPasswordSchema.safeParse({ password });
    if (!parsed.success) {
      setError(parsed.error.issues[0]?.message ?? "Periksa input.");
      return;
    }
    setBusy(true);
    try {
      await platformService.resetUserPassword(user.id, parsed.data.password);
      toast.success("Password berhasil direset");
      setPassword("");
      onClose();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "Gagal mereset password. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      open={!!user}
      onClose={() => {
        setPassword("");
        setError("");
        onClose();
      }}
      title="Reset Password"
      description={user ? `Atur ulang password untuk "${user.name}".` : ""}
    >
      <div className="grid gap-4">
        <div className="grid gap-2">
          <Label>Password Baru</Label>
          <Input
            type="password"
            value={password}
            onChange={(e) => {
              setPassword(e.target.value);
              if (error) setError("");
            }}
            placeholder="••••••••"
            aria-invalid={!!error}
          />
          <FieldError msg={error} />
        </div>
        <div className="flex justify-end">
          <Button loading={busy} onClick={submit}>
            Reset Password
          </Button>
        </div>
      </div>
    </Modal>
  );
}
