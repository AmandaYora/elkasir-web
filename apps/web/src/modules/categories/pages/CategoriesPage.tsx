import { useMemo, useState } from "react";
import { Search, Plus, MoreHorizontal, Pencil, Trash2, Tags } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
import { Badge } from "@/shared/components/ui/badge";
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
import { categoriesService } from "@/modules/categories/services/categories.service";
import { categorySchema } from "@/modules/categories/schemas/category.schema";
import type { Category } from "@/modules/categories/types/category.types";

const PAGE_SIZE = 10;

export default function CategoriesPage() {
  const categoriesQuery = useAsync(() => categoriesService.list({ limit: 200 }), []);
  const list = categoriesQuery.data?.data ?? [];

  const [q, setQ] = useState("");
  const [page, setPage] = useState(1);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Category | null>(null);
  const [deleting, setDeleting] = useState<Category | null>(null);
  const [busy, setBusy] = useState(false);

  const filtered = useMemo(
    () => list.filter((c) => q === "" || c.name.toLowerCase().includes(q.toLowerCase())),
    [list, q],
  );

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  const refresh = () => categoriesQuery.refetch();

  const remove = async () => {
    if (!deleting) return;
    if (deleting.productCount > 0) {
      toast.error(`Kategori masih dipakai ${deleting.productCount} produk.`);
      setDeleting(null);
      return;
    }
    setBusy(true);
    try {
      await categoriesService.remove(deleting.id);
      toast.success("Kategori berhasil dihapus");
      setDeleting(null);
      refresh();
    } catch (e) {
      toast.error("Gagal menghapus kategori. Coba lagi.");
      setDeleting(null);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">Kategori Produk</h2>
          <p className="text-sm text-muted">
            {list.length} kategori · dipakai untuk mengelompokkan produk
          </p>
        </div>
        <Button
          size="sm"
          onClick={() => {
            setEditing(null);
            setFormOpen(true);
          }}
        >
          <Plus className="h-4 w-4" /> Tambah Kategori
        </Button>
      </div>

      <Card>
        <CardContent className="p-4">
          <div className="flex flex-wrap items-center gap-2">
            <div className="relative min-w-[220px] flex-1">
              <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted" />
              <Input
                value={q}
                onChange={(e) => {
                  setQ(e.target.value);
                  setPage(1);
                }}
                placeholder="Cari kategori…"
                className="pl-9"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        {categoriesQuery.loading ? (
          <LoadingState />
        ) : categoriesQuery.error ? (
          <ErrorState message="Gagal memuat kategori. Coba lagi." onRetry={refresh} />
        ) : paged.length === 0 ? (
          <EmptyState title="Belum ada kategori" description="Tambahkan kategori pertama Anda." />
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Kategori</TableHead>
                  <TableHead>Jumlah Produk</TableHead>
                  <TableHead className="w-[60px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {paged.map((c) => (
                  <TableRow key={c.id}>
                    <TableCell>
                      <div className="flex items-center gap-2.5">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                          <Tags className="h-4 w-4" />
                        </div>
                        <span className="text-sm font-medium">{c.name}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge tone="neutral">{c.productCount} produk</Badge>
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
                        <DropdownItem danger onClick={() => setDeleting(c)}>
                          <Trash2 className="h-3.5 w-3.5" /> Hapus
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
              label={`Menampilkan ${paged.length} dari ${filtered.length} kategori`}
            />
          </>
        )}
      </Card>

      <Modal
        open={formOpen}
        onClose={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        title={editing ? "Ubah Kategori" : "Tambah Kategori"}
        description="Kategori dipakai untuk mengelompokkan produk di katalog & POS."
      >
        <CategoryForm
          key={editing?.id ?? "new"}
          editing={editing}
          existing={list}
          busy={busy}
          onSubmit={async (name) => {
            setBusy(true);
            try {
              if (editing) await categoriesService.update(editing.id, { name });
              else await categoriesService.create({ name });
              toast.success(
                editing ? "Kategori berhasil diperbarui" : "Kategori berhasil ditambahkan",
              );
              setFormOpen(false);
              setEditing(null);
              refresh();
            } catch (e) {
              toast.error("Gagal menyimpan kategori. Coba lagi.");
            } finally {
              setBusy(false);
            }
          }}
        />
      </Modal>

      <ConfirmDialog
        open={!!deleting}
        title="Hapus kategori ini?"
        description={
          deleting
            ? `Kategori "${deleting.name}" akan dihapus. Kategori yang masih dipakai produk tidak dapat dihapus.`
            : ""
        }
        confirmLabel="Hapus"
        danger
        loading={busy}
        onConfirm={remove}
        onClose={() => setDeleting(null)}
      />
    </div>
  );
}

function CategoryForm({
  editing,
  existing,
  busy,
  onSubmit,
}: {
  editing: Category | null;
  existing: Category[];
  busy: boolean;
  onSubmit: (name: string) => void;
}) {
  const [name, setName] = useState(editing?.name ?? "");
  const [error, setError] = useState("");

  const submit = () => {
    const value = name.trim();
    const parsed = categorySchema.safeParse({ name: value });
    if (!parsed.success) {
      setError(parsed.error.issues[0]?.message ?? "Periksa input.");
      return;
    }
    if (
      existing.some((c) => c.id !== editing?.id && c.name.toLowerCase() === value.toLowerCase())
    ) {
      setError("Kategori sudah ada");
      return;
    }
    setError("");
    onSubmit(value);
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label>Nama Kategori</Label>
        <Input
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            if (error) setError("");
          }}
          placeholder="mis. Minuman Dingin"
          onKeyDown={(e) => e.key === "Enter" && submit()}
          aria-invalid={!!error}
        />
        <FieldError msg={error} />
      </div>
      <div className="flex justify-end">
        <Button loading={busy} onClick={submit}>
          {editing ? "Simpan Perubahan" : "Tambah Kategori"}
        </Button>
      </div>
    </div>
  );
}
