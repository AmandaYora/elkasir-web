import { useMemo, useRef, useState } from "react";
import { Search, Plus, MoreHorizontal, Pencil, Trash2, Eye, Upload, X } from "lucide-react";
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
import { Drawer } from "@/shared/components/ui/drawer";
import { ConfirmDialog } from "@/shared/components/ui/confirm-dialog";
import { Pagination } from "@/shared/components/ui/pagination";
import { LoadingState, ErrorState, EmptyState } from "@/shared/components/feedback";
import { formatIDR } from "@/shared/lib/formatter";
import { useAsync } from "@/shared/hooks/useAsync";
import { productsService } from "@/modules/products/services/products.service";
import { mediaService } from "@/shared/services/media.service";
import { DEFAULT_PRODUCT_IMAGE_URL } from "@/shared/lib/image";
import { productSchema } from "@/modules/products/schemas/product.schema";
import { ProductStatusBadge } from "@/modules/products/components/ProductStatusBadge";
import type { Product, ProductInput, ProductStatus } from "@/modules/products/types/product.types";

const PAGE_SIZE = 10;

export default function ProductsPage() {
  const productsQuery = useAsync(() => productsService.list({ limit: 200 }), []);
  const categoriesQuery = useAsync(() => productsService.listCategories(), []);

  const items = productsQuery.data?.data ?? [];
  const categoryOptions = categoriesQuery.data?.data ?? [];

  const [q, setQ] = useState("");
  const [cat, setCat] = useState("all");
  const [status, setStatus] = useState("all");
  const [page, setPage] = useState(1);
  const [detail, setDetail] = useState<Product | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Product | null>(null);
  const [deleting, setDeleting] = useState<Product | null>(null);
  const [busy, setBusy] = useState(false);

  const distinctCats = useMemo(
    () => Array.from(new Set(items.map((p) => p.category).filter(Boolean))).sort(),
    [items],
  );

  const filtered = useMemo(() => {
    return items.filter(
      (p) =>
        (cat === "all" || p.category === cat) &&
        (status === "all" || p.status === status) &&
        (q === "" ||
          p.name.toLowerCase().includes(q.toLowerCase()) ||
          p.sku.toLowerCase().includes(q.toLowerCase())),
    );
  }, [items, q, cat, status]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  const refresh = () => productsQuery.refetch();

  const remove = async () => {
    if (!deleting) return;
    setBusy(true);
    try {
      await productsService.remove(deleting.id);
      toast.success("Produk berhasil dihapus");
      setDeleting(null);
      refresh();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Gagal menghapus produk");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 p-4 md:p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-text">Produk</h2>
          <p className="text-sm text-muted">{items.length} produk dalam katalog Anda</p>
        </div>
        <Button
          size="sm"
          onClick={() => {
            setEditing(null);
            setFormOpen(true);
          }}
        >
          <Plus className="h-4 w-4" /> Tambah Produk
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
                placeholder="Cari nama atau SKU…"
                className="pl-9"
              />
            </div>
            <Select
              value={cat}
              onChange={(e) => {
                setCat(e.target.value);
                setPage(1);
              }}
              className="w-[160px]"
            >
              <option value="all">Semua kategori</option>
              {distinctCats.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </Select>
            <Select
              value={status}
              onChange={(e) => {
                setStatus(e.target.value);
                setPage(1);
              }}
              className="w-[140px]"
            >
              <option value="all">Semua status</option>
              <option value="active">Aktif</option>
              <option value="inactive">Nonaktif</option>
            </Select>
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        {productsQuery.loading ? (
          <LoadingState />
        ) : productsQuery.error ? (
          <ErrorState message={productsQuery.error} onRetry={refresh} />
        ) : paged.length === 0 ? (
          <EmptyState title="Belum ada produk" description="Tambahkan produk pertama Anda." />
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Produk</TableHead>
                  <TableHead>SKU</TableHead>
                  <TableHead>Kategori</TableHead>
                  <TableHead className="text-right">Harga</TableHead>
                  <TableHead className="text-right">Stok</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-[60px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {paged.map((p) => (
                  <TableRow key={p.id} className="cursor-pointer" onClick={() => setDetail(p)}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <img
                          src={p.imageUrl || DEFAULT_PRODUCT_IMAGE_URL}
                          alt=""
                          loading="lazy"
                          onError={(e) => {
                            if (e.currentTarget.src !== DEFAULT_PRODUCT_IMAGE_URL)
                              e.currentTarget.src = DEFAULT_PRODUCT_IMAGE_URL;
                          }}
                          className="h-10 w-10 shrink-0 rounded-md border border-border object-cover"
                        />
                        <div>
                          <div className="font-medium">{p.name}</div>
                          <div className="text-xs text-muted">Modal {formatIDR(p.cost)}</div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted">{p.sku}</TableCell>
                    <TableCell className="text-sm">{p.category || "—"}</TableCell>
                    <TableCell className="text-right font-medium">{formatIDR(p.price)}</TableCell>
                    <TableCell className="text-right">
                      <span
                        className={
                          p.stock === 0 ? "text-danger" : p.stock < 10 ? "text-warning" : ""
                        }
                      >
                        {p.stock}
                      </span>
                    </TableCell>
                    <TableCell>
                      <ProductStatusBadge status={p.status} />
                    </TableCell>
                    <TableCell onClick={(e) => e.stopPropagation()}>
                      <Dropdown
                        trigger={
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        }
                      >
                        <DropdownItem onClick={() => setDetail(p)}>
                          <Eye className="h-3.5 w-3.5" /> Lihat
                        </DropdownItem>
                        <DropdownItem
                          onClick={() => {
                            setEditing(p);
                            setFormOpen(true);
                          }}
                        >
                          <Pencil className="h-3.5 w-3.5" /> Ubah
                        </DropdownItem>
                        <DropdownItem danger onClick={() => setDeleting(p)}>
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
              label={`Menampilkan ${paged.length} dari ${filtered.length} produk`}
            />
          </>
        )}
      </Card>

      {/* Detail drawer */}
      <Drawer
        open={!!detail}
        onClose={() => setDetail(null)}
        title={detail?.name}
        description={detail?.sku}
      >
        {detail && (
          <div className="space-y-5">
            <img
              src={detail.imageUrl || DEFAULT_PRODUCT_IMAGE_URL}
              alt={detail.name}
              onError={(e) => {
                if (e.currentTarget.src !== DEFAULT_PRODUCT_IMAGE_URL)
                  e.currentTarget.src = DEFAULT_PRODUCT_IMAGE_URL;
              }}
              className="aspect-video w-full rounded-lg border border-border object-cover"
            />
            <div className="grid grid-cols-2 gap-3">
              {[
                ["Kategori", detail.category || "—"],
                ["Status", detail.status === "active" ? "Aktif" : "Nonaktif"],
                ["Harga jual", formatIDR(detail.price)],
                ["Modal", formatIDR(detail.cost)],
                ["Stok saat ini", `${detail.stock} unit`],
              ].map(([k, v]) => (
                <div key={k} className="rounded-lg border border-border bg-surface-muted p-3">
                  <div className="text-[11px] uppercase tracking-wider text-muted">{k}</div>
                  <div className="mt-1 text-sm font-medium">{v}</div>
                </div>
              ))}
            </div>
            <StockAdjuster
              product={detail}
              onDone={(updated) => {
                setDetail(updated);
                refresh();
              }}
            />
            <Button
              className="w-full"
              size="sm"
              onClick={() => {
                setEditing(detail);
                setDetail(null);
                setFormOpen(true);
              }}
            >
              <Pencil className="h-3.5 w-3.5" /> Ubah produk
            </Button>
          </div>
        )}
      </Drawer>

      {/* Create / edit form */}
      <Modal
        open={formOpen}
        onClose={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        title={editing ? "Ubah Produk" : "Tambah Produk"}
        description="Lengkapi detail produk pada katalog Anda."
      >
        <ProductForm
          key={editing?.id ?? "new"}
          editing={editing}
          categories={categoryOptions}
          onDone={() => {
            setFormOpen(false);
            setEditing(null);
            refresh();
          }}
        />
      </Modal>

      <ConfirmDialog
        open={!!deleting}
        title="Hapus produk ini?"
        description={deleting ? `"${deleting.name}" akan dihapus permanen dari katalog.` : ""}
        confirmLabel="Hapus"
        danger
        loading={busy}
        onConfirm={remove}
        onClose={() => setDeleting(null)}
      />
    </div>
  );
}

function StockAdjuster({ product, onDone }: { product: Product; onDone: (p: Product) => void }) {
  const [delta, setDelta] = useState(0);
  const [busy, setBusy] = useState(false);
  const apply = async () => {
    setBusy(true);
    try {
      const updated = await productsService.adjustStock(product.id, delta);
      toast.success("Stok diperbarui");
      setDelta(0);
      onDone(updated);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Gagal menyesuaikan stok");
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="rounded-lg border border-border p-3">
      <div className="text-[11px] uppercase tracking-wider text-muted">Sesuaikan stok</div>
      <div className="mt-2 flex items-center gap-2">
        <Input
          type="number"
          value={delta}
          onChange={(e) => setDelta(Math.trunc(+e.target.value))}
          placeholder="mis. 10 atau -5"
        />
        <Button size="sm" loading={busy} disabled={delta === 0} onClick={apply}>
          Terapkan
        </Button>
      </div>
      <p className="mt-1 text-xs text-muted">Nilai positif menambah, negatif mengurangi stok.</p>
    </div>
  );
}

function ProductForm({
  editing,
  categories,
  onDone,
}: {
  editing: Product | null;
  categories: { id: string; name: string }[];
  onDone: () => void;
}) {
  const [form, setForm] = useState<ProductInput>({
    name: editing?.name ?? "",
    sku: editing?.sku ?? "",
    categoryId: editing?.categoryId,
    price: editing?.price ?? 0,
    cost: editing?.cost ?? 0,
    stock: editing?.stock ?? 0,
    status: (editing?.status as ProductStatus) ?? "active",
    imageUrl: editing?.imageUrl ?? "",
  });
  const [busy, setBusy] = useState(false);
  const [imageBusy, setImageBusy] = useState(false);

  const submit = async () => {
    const parsed = productSchema.safeParse(form);
    if (!parsed.success) {
      toast.error(parsed.error.issues[0]?.message ?? "Periksa input.");
      return;
    }
    setBusy(true);
    try {
      if (editing) await productsService.update(editing.id, form);
      else await productsService.create(form);
      toast.success(editing ? "Produk berhasil diperbarui" : "Produk berhasil ditambahkan");
      onDone();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Gagal menyimpan produk");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label>Nama produk</Label>
        <Input
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          placeholder="Nasi Ayam"
        />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="grid gap-2">
          <Label>SKU</Label>
          <Input
            value={form.sku}
            onChange={(e) => setForm({ ...form, sku: e.target.value })}
            placeholder="SKU-1234"
          />
        </div>
        <div className="grid gap-2">
          <Label>Kategori</Label>
          <Select
            value={form.categoryId ?? "none"}
            onChange={(e) =>
              setForm({
                ...form,
                categoryId: e.target.value === "none" ? undefined : e.target.value,
              })
            }
          >
            <option value="none">Tanpa kategori</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </Select>
        </div>
      </div>
      <div className="grid grid-cols-3 gap-3">
        <div className="grid gap-2">
          <Label>Harga</Label>
          <Input
            type="number"
            value={form.price}
            onChange={(e) => setForm({ ...form, price: +e.target.value })}
          />
        </div>
        <div className="grid gap-2">
          <Label>Modal</Label>
          <Input
            type="number"
            value={form.cost}
            onChange={(e) => setForm({ ...form, cost: +e.target.value })}
          />
        </div>
        <div className="grid gap-2">
          <Label>Stok</Label>
          <Input
            type="number"
            value={form.stock}
            onChange={(e) => setForm({ ...form, stock: +e.target.value })}
          />
        </div>
      </div>
      <div className="grid gap-2">
        <Label>Status</Label>
        <Select
          value={form.status}
          onChange={(e) => setForm({ ...form, status: e.target.value as ProductStatus })}
        >
          <option value="active">Aktif</option>
          <option value="inactive">Nonaktif</option>
        </Select>
      </div>
      <ImageUploadField
        value={form.imageUrl ?? ""}
        uploadingChange={setImageBusy}
        onChange={(url) => setForm((f) => ({ ...f, imageUrl: url }))}
      />
      <div className="flex justify-end">
        <Button loading={busy} disabled={imageBusy} onClick={submit}>
          {editing ? "Simpan Perubahan" : "Tambah Produk"}
        </Button>
      </div>
    </div>
  );
}

function ImageUploadField({
  value,
  onChange,
  uploadingChange,
}: {
  value: string;
  onChange: (url: string) => void;
  uploadingChange: (busy: boolean) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);

  const onFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = ""; // izinkan memilih file yang sama lagi
    if (!file) return;
    if (!file.type.startsWith("image/")) {
      toast.error("File harus berupa gambar.");
      return;
    }
    setUploading(true);
    uploadingChange(true);
    setProgress(0);
    try {
      const res = await mediaService.uploadImage(file, "product", setProgress);
      onChange(res.url);
      toast.success("Gambar berhasil diunggah");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal mengunggah gambar");
    } finally {
      setUploading(false);
      uploadingChange(false);
    }
  };

  return (
    <div className="grid gap-2">
      <Label>Gambar produk</Label>
      <div className="flex items-center gap-3">
        <div className="relative h-20 w-20 shrink-0 overflow-hidden rounded-lg border border-border bg-surface-muted">
          <img
            src={value || DEFAULT_PRODUCT_IMAGE_URL}
            alt=""
            onError={(e) => {
              if (e.currentTarget.src !== DEFAULT_PRODUCT_IMAGE_URL)
                e.currentTarget.src = DEFAULT_PRODUCT_IMAGE_URL;
            }}
            className="h-full w-full object-cover"
          />
          {uploading && (
            <div className="absolute inset-0 flex items-center justify-center bg-black/50 text-xs font-medium text-white">
              {progress}%
            </div>
          )}
        </div>
        <div className="flex flex-col items-start gap-1.5">
          <input ref={inputRef} type="file" accept="image/*" className="hidden" onChange={onFile} />
          <Button
            type="button"
            variant="outline"
            size="sm"
            loading={uploading}
            onClick={() => inputRef.current?.click()}
          >
            <Upload className="h-3.5 w-3.5" /> {value ? "Ganti gambar" : "Unggah gambar"}
          </Button>
          {value && !uploading && (
            <Button type="button" variant="ghost" size="sm" onClick={() => onChange("")}>
              <X className="h-3.5 w-3.5" /> Hapus gambar
            </Button>
          )}
        </div>
      </div>
      <p className="text-xs text-muted">
        JPG/PNG/WebP. Dikompres otomatis (browser + server) sebelum disimpan.
      </p>
    </div>
  );
}
