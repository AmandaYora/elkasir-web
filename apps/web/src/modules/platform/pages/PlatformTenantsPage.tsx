import { useMemo, useState } from "react";
import { Search, Plus, MoreHorizontal, Ban, CheckCircle2, Building2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
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
import { FieldError } from "@/shared/components/ui/field-error";
import { ConfirmDialog } from "@/shared/components/ui/confirm-dialog";
import { Pagination } from "@/shared/components/ui/pagination";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { useAsync } from "@/shared/hooks/useAsync";
import { zodFieldErrors } from "@/shared/lib/form";
import { platformService } from "@/modules/platform/services/platform.service";
import { createTenantSchema } from "@/modules/platform/schemas/tenant.schema";
import { TenantStatusBadge } from "@/modules/platform/components/TenantStatusBadge";
import type { Tenant, CreateTenantInput } from "@/modules/platform/types/platform.types";

const PAGE_SIZE = 10;

export default function PlatformTenantsPage() {
  const tenantsQuery = useAsync(() => platformService.listTenants(), []);
  const list = tenantsQuery.data ?? [];

  const [q, setQ] = useState("");
  const [page, setPage] = useState(1);
  const [formOpen, setFormOpen] = useState(false);
  const [toggling, setToggling] = useState<Tenant | null>(null);
  const [busy, setBusy] = useState(false);

  const filtered = useMemo(
    () =>
      list.filter(
        (t) =>
          q === "" ||
          t.name.toLowerCase().includes(q.toLowerCase()) ||
          t.slug.toLowerCase().includes(q.toLowerCase()),
      ),
    [list, q],
  );

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  const refresh = () => tenantsQuery.refetch();

  const toggleStatus = async () => {
    if (!toggling) return;
    const nextStatus = toggling.status === "active" ? "suspended" : "active";
    setBusy(true);
    try {
      await platformService.setTenantStatus(toggling.id, nextStatus);
      toast.success(
        nextStatus === "suspended" ? "Tenant berhasil dinonaktifkan" : "Tenant berhasil diaktifkan",
      );
      setToggling(null);
      refresh();
    } catch {
      toast.error("Gagal mengubah status tenant. Coba lagi.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">Tenant</h2>
          <p className="text-sm text-muted">{list.length} tenant terdaftar</p>
        </div>
        <Button size="sm" onClick={() => setFormOpen(true)}>
          <Plus className="h-4 w-4" /> Tambah Tenant
        </Button>
      </div>

      <Card>
        <CardContent className="p-4">
          <div className="relative min-w-[220px] flex-1">
            <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted" />
            <Input
              value={q}
              onChange={(e) => {
                setQ(e.target.value);
                setPage(1);
              }}
              placeholder="Cari nama atau slug tenant…"
              className="pl-9"
            />
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        {tenantsQuery.loading ? (
          <LoadingState />
        ) : tenantsQuery.error ? (
          <ErrorState message="Gagal memuat tenant. Coba lagi." onRetry={refresh} />
        ) : paged.length === 0 ? (
          <EmptyState title="Belum ada tenant" description="Tambahkan tenant pertama Anda." />
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Tenant</TableHead>
                  <TableHead>Slug</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Dibuat</TableHead>
                  <TableHead className="w-[60px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {paged.map((t) => (
                  <TableRow key={t.id}>
                    <TableCell>
                      <div className="flex items-center gap-2.5">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                          <Building2 className="h-4 w-4" />
                        </div>
                        <span className="text-sm font-medium">{t.name}</span>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm text-muted">{t.slug}</TableCell>
                    <TableCell>
                      <TenantStatusBadge status={t.status} />
                    </TableCell>
                    <TableCell className="text-sm text-muted">
                      {new Date(t.createdAt).toLocaleDateString("id-ID")}
                    </TableCell>
                    <TableCell>
                      <Dropdown
                        trigger={
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        }
                      >
                        <DropdownItem danger={t.status === "active"} onClick={() => setToggling(t)}>
                          {t.status === "active" ? (
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
                ))}
              </TableBody>
            </Table>
            <Pagination
              page={page}
              totalPages={totalPages}
              total={filtered.length}
              onPageChange={setPage}
              label={`Menampilkan ${paged.length} dari ${filtered.length} tenant`}
            />
          </>
        )}
      </Card>

      <Modal
        open={formOpen}
        onClose={() => setFormOpen(false)}
        title="Tambah Tenant"
        description="Membuat toko baru sekaligus akun pemilik pertamanya."
      >
        <TenantForm
          onDone={() => {
            setFormOpen(false);
            refresh();
          }}
        />
      </Modal>

      <ConfirmDialog
        open={!!toggling}
        title={toggling?.status === "active" ? "Nonaktifkan tenant ini?" : "Aktifkan tenant ini?"}
        description={
          toggling
            ? toggling.status === "active"
              ? `"${toggling.name}" akan kehilangan akses ke web dan POS seketika.`
              : `"${toggling.name}" akan mendapatkan kembali akses seketika.`
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

function TenantForm({ onDone }: { onDone: () => void }) {
  const [form, setForm] = useState<CreateTenantInput>({
    storeName: "",
    storeSlug: "",
    ownerName: "",
    ownerEmail: "",
    ownerPassword: "",
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

  const setField = <K extends keyof CreateTenantInput>(key: K, value: CreateTenantInput[K]) => {
    setForm((f) => ({ ...f, [key]: value }));
    setErrors((e) => (e[key] ? { ...e, [key]: "" } : e));
  };

  const submit = async () => {
    const parsed = createTenantSchema.safeParse(form);
    if (!parsed.success) {
      setErrors(zodFieldErrors(parsed.error));
      return;
    }
    setBusy(true);
    try {
      await platformService.createTenant(parsed.data);
      toast.success("Tenant berhasil dibuat");
      onDone();
    } catch {
      toast.error("Gagal membuat tenant. Slug atau email pemilik mungkin sudah dipakai.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label>Nama Toko</Label>
        <Input
          value={form.storeName}
          onChange={(e) => setField("storeName", e.target.value)}
          placeholder="mis. Warkop Budi"
          aria-invalid={!!errors.storeName}
        />
        <FieldError msg={errors.storeName} />
      </div>
      <div className="grid gap-2">
        <Label>Slug</Label>
        <Input
          value={form.storeSlug}
          onChange={(e) => setField("storeSlug", e.target.value.toLowerCase())}
          placeholder="mis. warkop-budi"
          aria-invalid={!!errors.storeSlug}
        />
        <FieldError msg={errors.storeSlug} />
      </div>
      <div className="grid gap-2">
        <Label>Nama Pemilik</Label>
        <Input
          value={form.ownerName}
          onChange={(e) => setField("ownerName", e.target.value)}
          placeholder="mis. Budi Santoso"
          aria-invalid={!!errors.ownerName}
        />
        <FieldError msg={errors.ownerName} />
      </div>
      <div className="grid gap-2">
        <Label>Email Pemilik</Label>
        <Input
          type="email"
          value={form.ownerEmail}
          onChange={(e) => setField("ownerEmail", e.target.value)}
          placeholder="budi@contoh.com"
          aria-invalid={!!errors.ownerEmail}
        />
        <FieldError msg={errors.ownerEmail} />
      </div>
      <div className="grid gap-2">
        <Label>Password Pemilik</Label>
        <Input
          type="password"
          value={form.ownerPassword}
          onChange={(e) => setField("ownerPassword", e.target.value)}
          placeholder="••••••••"
          aria-invalid={!!errors.ownerPassword}
        />
        <FieldError msg={errors.ownerPassword} />
      </div>
      <div className="flex justify-end">
        <Button loading={busy} onClick={submit}>
          Buat Tenant
        </Button>
      </div>
    </div>
  );
}
