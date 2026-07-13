import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { toast } from "sonner";
import {
  Search,
  Plus,
  Minus,
  Trash2,
  ShoppingBag,
  ArrowLeft,
  QrCode,
  ScanLine,
  Utensils,
  Loader2,
} from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { formatIDR } from "@/shared/lib/formatter";
import { DEFAULT_PRODUCT_IMAGE_URL } from "@/shared/lib/image";
import { ApiError } from "@/shared/types/api";
import {
  QrisPaymentPanel,
  CashierBarcodePanel,
  OrderBreakdown,
} from "@/modules/self-order/components/SelfOrderPayment";
import { publicOrderService } from "@/modules/self-order/services/public-order.service";
import type {
  PlaceResult,
  PublicMenu,
  PublicSelfOrderStatus,
  QuoteResult,
} from "@/modules/self-order/types/self-order.types";

type Step = "menu" | "review" | "qris" | "cashier";

const errMsg = (_e: unknown) => "Pesanan belum bisa diproses. Coba lagi.";

// Halaman self-order pelanggan (publik, tanpa auth). Slug toko + kode meja diambil dari rute.
export default function PublicOrderPage() {
  const { slug = "", code = "" } = useParams();

  const [menu, setMenu] = useState<PublicMenu | null>(null);
  const [menuLoading, setMenuLoading] = useState(true);
  const [menuError, setMenuError] = useState<unknown>(null);

  const [step, setStep] = useState<Step>("menu");
  const [cat, setCat] = useState("Semua");
  const [q, setQ] = useState("");
  const [cart, setCart] = useState<Record<string, number>>({});
  const [note, setNote] = useState("");
  const [placed, setPlaced] = useState<PlaceResult | null>(null);
  const [placing, setPlacing] = useState(false);
  const [orderStatus, setOrderStatus] = useState<PublicSelfOrderStatus | null>(null);

  // Ambil menu meja saat mount / kode meja berubah.
  useEffect(() => {
    let active = true;
    setMenuLoading(true);
    setMenuError(null);
    publicOrderService
      .menu(slug, code)
      .then((data) => {
        if (active) setMenu(data);
      })
      .catch((e) => {
        if (active) setMenuError(e);
      })
      .finally(() => {
        if (active) setMenuLoading(false);
      });
    return () => {
      active = false;
    };
  }, [slug, code]);

  const products = menu?.products ?? [];
  const categories = useMemo(() => ["Semua", ...(menu?.categories ?? [])], [menu]);

  // Flag metode bayar dari settings toko (default ON agar API lama tetap kompatibel).
  const selfOrderEnabled = menu?.featureSelfOrder ?? true;
  const qrisEnabled = menu?.featureQris ?? true;
  const cashierEnabled = menu?.featurePayAtCashier ?? true;

  const visible = useMemo(() => {
    const query = q.trim().toLowerCase();
    return products.filter(
      (p) =>
        (cat === "Semua" || p.category === cat) &&
        (query === "" ||
          p.name.toLowerCase().includes(query) ||
          p.category.toLowerCase().includes(query)),
    );
  }, [products, cat, q]);

  const lines = useMemo(
    () =>
      Object.entries(cart)
        .map(([id, qty]) => ({ product: products.find((p) => p.id === id)!, qty }))
        .filter((l) => l.product && l.qty > 0),
    [cart, products],
  );
  const totalItems = lines.reduce((s, l) => s + l.qty, 0);
  const total = lines.reduce((s, l) => s + l.product.price * l.qty, 0);

  // Rincian biaya (skenario QRIS) untuk ditampilkan di langkah review sebelum pesanan dibuat.
  // Angka final & otoritatif tetap diambil dari pesanan yang sudah dibuat (placed.order).
  const [quote, setQuote] = useState<QuoteResult | null>(null);
  useEffect(() => {
    if (step !== "review" || lines.length === 0) {
      setQuote(null);
      return;
    }
    let active = true;
    // Tampilkan rincian utk metode yang aktif: QRIS (termasuk biaya gateway) bila tersedia,
    // jika tidak, jalur tunai (tanpa biaya gateway).
    const quoteMethod = qrisEnabled ? "qris" : "cash";
    const timer = setTimeout(() => {
      publicOrderService
        .quote(slug, code, {
          items: lines.map((l) => ({ productId: l.product.id, quantity: l.qty, note: "" })),
          paymentMethod: quoteMethod,
        })
        .then((q) => active && setQuote(q))
        .catch(() => active && setQuote(null));
    }, 250);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [step, lines, slug, code, qrisEnabled]);

  const add = (id: string) => setCart((c) => ({ ...c, [id]: (c[id] ?? 0) + 1 }));
  const dec = (id: string) =>
    setCart((c) => {
      const next = (c[id] ?? 0) - 1;
      const copy = { ...c };
      if (next <= 0) delete copy[id];
      else copy[id] = next;
      return copy;
    });
  const remove = (id: string) =>
    setCart((c) => {
      const copy = { ...c };
      delete copy[id];
      return copy;
    });

  const place = async (paymentMethod: "qris" | "cash") => {
    setPlacing(true);
    try {
      const res = await publicOrderService.place(slug, code, {
        items: lines.map((l) => ({ productId: l.product.id, quantity: l.qty, note: "" })),
        paymentMethod,
        customerNote: note.trim(),
      });
      setPlaced(res);
      setOrderStatus(null);
      setStep(paymentMethod === "qris" ? "qris" : "cashier");
    } catch (e) {
      toast.error(errMsg(e));
    } finally {
      setPlacing(false);
    }
  };

  // Status QRIS via Server-Sent Events — layar maju OTOMATIS saat callback gateway menandai
  // lunas. Tidak ada polling: satu koneksi event yang di-push server (lihat subscribeStatus).
  const orderId = placed?.order.id;
  const qrisPaid = orderStatus?.paymentStatus === "paid";

  useEffect(() => {
    if (step !== "qris" || !orderId) return;
    return publicOrderService.subscribeStatus(orderId, setOrderStatus);
  }, [step, orderId]);

  const simulatePaid = async () => {
    if (!orderId) return;
    try {
      await publicOrderService.simulatePaid(orderId);
      const s = await publicOrderService.status(orderId);
      setOrderStatus(s);
    } catch (e) {
      toast.error(errMsg(e));
    }
  };

  const reset = () => {
    setCart({});
    setNote("");
    setPlaced(null);
    setOrderStatus(null);
    setStep("menu");
  };

  if (menuLoading) {
    return (
      <div className="flex min-h-dvh items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted" />
      </div>
    );
  }

  if (menuError || !menu) {
    const notFound = menuError instanceof ApiError && menuError.status === 404;
    return (
      <div className="auth-rise mx-auto flex min-h-dvh max-w-md flex-col items-center justify-center gap-3 p-6 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-danger-soft text-danger">
          <Utensils className="h-6 w-6" />
        </div>
        <h1 className="font-display text-lg font-bold">Meja tidak dikenali</h1>
        <p className="text-sm text-muted">
          {notFound
            ? `QR untuk kode meja "${code}" tidak ditemukan. Silakan hubungi staf.`
            : "Menu belum bisa dimuat. Coba lagi atau hubungi staf."}
        </p>
      </div>
    );
  }

  if (menu.table.status !== "active") {
    return (
      <div className="auth-rise mx-auto flex min-h-dvh max-w-md flex-col items-center justify-center gap-3 p-6 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-surface-muted text-muted">
          <Utensils className="h-6 w-6" />
        </div>
        <h1 className="font-display text-lg font-bold">Meja belum tersedia</h1>
        <p className="text-sm text-muted">
          Meja {menu.table.name} sedang tidak menerima pesanan via QR. Silakan hubungi staf.
        </p>
      </div>
    );
  }

  if (!selfOrderEnabled) {
    return (
      <div className="auth-rise mx-auto flex min-h-dvh max-w-md flex-col items-center justify-center gap-3 p-6 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-surface-muted text-muted">
          <Utensils className="h-6 w-6" />
        </div>
        <h1 className="font-display text-lg font-bold">Pemesanan mandiri ditutup</h1>
        <p className="text-sm text-muted">
          Pemesanan via QR sedang tidak tersedia untuk saat ini. Silakan pesan langsung ke staf.
        </p>
      </div>
    );
  }

  // Total sudah dijelaskan di tiket/OrderBreakdown di atasnya — kartu ini fokus ke "apa
  // yang dipesan" saja (item + catatan), tanpa mengulang baris Total sekali lagi.
  const OrderSummary = () => {
    if (!placed) return null;
    const o = placed.order;
    return (
      <div className="w-full rounded-2xl border border-border bg-surface text-left">
        <div className="border-b border-border px-4 py-2.5 text-xs font-semibold uppercase tracking-wider text-muted">
          Ringkasan pesanan
        </div>
        <div className="divide-y divide-border">
          {o.items.map((it, i) => (
            <div key={i} className="flex items-center justify-between px-4 py-2.5 text-sm">
              <span>
                {it.quantity} × {it.productName}
              </span>
              <span className="font-medium tabular-nums">{formatIDR(it.lineTotal)}</span>
            </div>
          ))}
        </div>
        {o.customerNote && (
          <div className="border-t border-border px-4 py-2.5 text-xs text-muted">
            Catatan: {o.customerNote}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="mx-auto min-h-dvh max-w-md bg-surface-muted pb-[calc(7rem+env(safe-area-inset-bottom))]">
      <div className="sticky top-0 z-10 border-b border-border bg-surface/95 px-4 py-3 backdrop-blur">
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm">
            <Utensils className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <div className="font-display text-sm font-extrabold leading-tight">Elkasir</div>
            <div className="text-xs text-muted">
              Pesan dari Meja <span className="font-semibold text-text">{menu.table.name}</span>
            </div>
          </div>
        </div>
        {/* Rel progres 3 langkah (Menu → Periksa → Bayar): posisi kiri-ke-kanan sendiri
            sudah menyampaikan urutan, jadi cukup diisi warnanya, tanpa perlu angka besar. */}
        <div className="mt-3">
          <div className="flex items-center gap-1.5">
            {(["menu", "review", "pay"] as const).map((seg, i) => {
              const current = step === "menu" ? 0 : step === "review" ? 1 : 2;
              return (
                <span
                  key={seg}
                  className={`h-1 flex-1 rounded-full transition-colors duration-300 ${
                    current >= i ? "bg-primary" : "bg-surface-muted"
                  }`}
                />
              );
            })}
          </div>
          <div className="mt-1.5 flex items-center justify-between text-[11px] font-medium">
            {["Menu", "Periksa", "Bayar"].map((label, i) => {
              const current = step === "menu" ? 0 : step === "review" ? 1 : 2;
              return (
                <span key={label} className={current >= i ? "text-text" : "text-muted"}>
                  {label}
                </span>
              );
            })}
          </div>
        </div>
      </div>

      {step === "menu" && (
        // Fragment (bukan div beranimasi) di level ini dengan sengaja: keyframe auth-rise
        // memakai transform, dan transform pada leluhur mana pun membuat descendant
        // position:fixed (bar keranjang di bawah) jadi terpaku ke KOTAK leluhur itu, bukan
        // ke viewport — bar bisa "hilang" tergulir ke bawah daftar menu yang panjang.
        // Jadi auth-rise ditaruh di div konten yang bisa discroll, bar tetap sibling di luar.
        <>
          <div key="menu" className="auth-rise">
            <div className="space-y-3 p-4">
              <div className="relative">
                <Search className="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
                <Input
                  value={q}
                  onChange={(e) => setQ(e.target.value)}
                  placeholder="Cari menu..."
                  className="h-11 rounded-xl pl-10"
                />
              </div>
              <div className="flex gap-2 overflow-x-auto pb-1">
                {categories.map((c) => (
                  <button
                    key={c}
                    onClick={() => setCat(c)}
                    className={`shrink-0 rounded-full border px-3.5 py-1.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 ${
                      cat === c
                        ? "border-primary bg-primary/10 text-primary"
                        : "border-border bg-surface text-muted hover:border-muted/60"
                    }`}
                  >
                    {c}
                  </button>
                ))}
              </div>
            </div>

            <div className="space-y-2 px-4">
              {visible.map((p) => {
                const qty = cart[p.id] ?? 0;
                return (
                  <div
                    key={p.id}
                    className="flex items-center gap-3 rounded-2xl border border-border bg-surface p-3 shadow-xs"
                  >
                    <img
                      src={p.imageUrl || DEFAULT_PRODUCT_IMAGE_URL}
                      alt={p.name}
                      loading="lazy"
                      onError={(e) => {
                        // Hindari loop bila gambar default sendiri gagal dimuat.
                        if (e.currentTarget.src !== DEFAULT_PRODUCT_IMAGE_URL) {
                          e.currentTarget.src = DEFAULT_PRODUCT_IMAGE_URL;
                        }
                      }}
                      className="h-14 w-14 shrink-0 rounded-xl bg-surface-muted object-cover"
                    />
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-semibold">{p.name}</div>
                      <div className="text-xs text-muted">{p.category}</div>
                      <div className="mt-0.5 text-sm font-bold tabular-nums">
                        {formatIDR(p.price)}
                      </div>
                    </div>
                    {qty === 0 ? (
                      <Button
                        size="sm"
                        variant="outline"
                        className="shrink-0 gap-1 active:scale-95"
                        onClick={() => add(p.id)}
                      >
                        <Plus className="h-3.5 w-3.5" /> Tambah
                      </Button>
                    ) : (
                      <div className="flex shrink-0 items-center gap-1.5 rounded-full bg-surface-muted p-1">
                        <Button
                          size="icon"
                          variant="ghost"
                          className="h-7 w-7 rounded-full bg-surface active:scale-90"
                          onClick={() => dec(p.id)}
                        >
                          <Minus className="h-3.5 w-3.5" />
                        </Button>
                        <span className="w-5 text-center text-sm font-bold tabular-nums">
                          {qty}
                        </span>
                        <Button
                          size="icon"
                          variant="ghost"
                          className="h-7 w-7 rounded-full bg-surface active:scale-90"
                          onClick={() => add(p.id)}
                        >
                          <Plus className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    )}
                  </div>
                );
              })}
              {visible.length === 0 && (
                <div className="py-12 text-center text-sm text-muted">Menu tidak ditemukan.</div>
              )}
            </div>
          </div>

          {/* Sibling di luar div auth-rise di atas (bukan child) — lihat catatan di pembuka
              blok "menu". auth-rise aman dipasang di elemen fixed ini SENDIRI (transform pada
              diri sendiri tidak memengaruhi posisi fixed-nya sendiri), hanya berbahaya bila
              dipasang di LELUHURNYA. */}
          {totalItems > 0 && (
            <div
              className="auth-rise fixed inset-x-0 bottom-0 z-10 mx-auto max-w-md p-4"
              style={{ paddingBottom: "max(1rem, env(safe-area-inset-bottom))" }}
            >
              <Button
                className="h-12 w-full justify-between gap-2 shadow-lg active:scale-[0.99]"
                onClick={() => setStep("review")}
              >
                <span className="flex items-center gap-2">
                  <ShoppingBag className="h-4 w-4" /> {totalItems} item
                </span>
                <span className="font-mono tabular-nums">Lanjut · {formatIDR(total)}</span>
              </Button>
            </div>
          )}
        </>
      )}

      {step === "review" && (
        <div key="review" className="auth-rise space-y-4 p-4">
          <button
            onClick={() => setStep("menu")}
            className="flex items-center gap-1 rounded-md text-sm text-muted transition-colors hover:text-text focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
          >
            <ArrowLeft className="h-4 w-4" /> Tambah menu lagi
          </button>

          <div className="rounded-2xl border border-border bg-surface">
            <div className="border-b border-border px-4 py-2.5 text-xs font-semibold uppercase tracking-wider text-muted">
              Pesanan Anda
            </div>
            <div className="divide-y divide-border">
              {lines.map((l) => (
                <div key={l.product.id} className="flex items-center gap-2 px-4 py-3">
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium">{l.product.name}</div>
                    <div className="text-xs text-muted tabular-nums">
                      {formatIDR(l.product.price)}
                    </div>
                  </div>
                  <div className="flex shrink-0 items-center gap-1 rounded-full bg-surface-muted p-1">
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-6 w-6 rounded-full bg-surface active:scale-90"
                      onClick={() => dec(l.product.id)}
                    >
                      <Minus className="h-3 w-3" />
                    </Button>
                    <span className="w-5 text-center text-sm font-bold tabular-nums">{l.qty}</span>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-6 w-6 rounded-full bg-surface active:scale-90"
                      onClick={() => add(l.product.id)}
                    >
                      <Plus className="h-3 w-3" />
                    </Button>
                  </div>
                  <span className="w-20 shrink-0 text-right text-sm font-semibold tabular-nums">
                    {formatIDR(l.product.price * l.qty)}
                  </span>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="h-7 w-7 shrink-0 text-danger hover:bg-danger-soft"
                    onClick={() => remove(l.product.id)}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ))}
            </div>
          </div>

          <div className="grid gap-2">
            <label className="text-sm font-medium">Catatan (opsional)</label>
            <Input
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="mis. tidak pedas, tanpa es"
              className="h-11 rounded-xl"
            />
          </div>

          {quote ? (
            <OrderBreakdown
              subtotal={quote.subtotal}
              serviceLine={quote.serviceLine}
              tax={quote.tax}
              total={quote.total}
              servicePercent={menu.servicePercent}
              taxPercent={menu.taxEnabled ? menu.taxPercent : undefined}
            />
          ) : (
            <div className="flex items-center justify-between rounded-2xl border border-border bg-surface px-4 py-3">
              <span className="text-sm text-muted">Subtotal</span>
              <span className="font-mono text-lg font-bold tabular-nums">{formatIDR(total)}</span>
            </div>
          )}

          <div className="space-y-2">
            <div className="px-1 text-sm font-medium">
              {qrisEnabled && cashierEnabled ? "Pilih cara pembayaran" : "Cara pembayaran"}
            </div>
            <p className="px-1 text-xs text-muted">
              Layanan & pajak (bila ada) ditambahkan ke total. Biaya QRIS hanya berlaku untuk
              pembayaran QRIS.
            </p>
            {qrisEnabled && (
              <button
                onClick={() => place("qris")}
                disabled={placing}
                className="flex w-full items-center gap-3 rounded-2xl border border-border bg-surface p-4 text-left transition-colors hover:border-primary hover:bg-primary/5 active:scale-[0.99] disabled:opacity-60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  <QrCode className="h-5 w-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-semibold">Bayar QRIS</div>
                  <div className="text-xs text-muted">
                    Scan QR & bayar dari ponsel. Pesanan langsung masuk setelah lunas.
                  </div>
                </div>
              </button>
            )}
            {cashierEnabled && (
              <button
                onClick={() => place("cash")}
                disabled={placing}
                className="flex w-full items-center gap-3 rounded-2xl border border-border bg-surface p-4 text-left transition-colors hover:border-primary hover:bg-primary/5 active:scale-[0.99] disabled:opacity-60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  <ScanLine className="h-5 w-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-semibold">Bayar di kasir</div>
                  <div className="text-xs text-muted">
                    Dapat barcode, tunjukkan ke kasir, lalu bayar tunai.
                  </div>
                </div>
              </button>
            )}
          </div>
        </div>
      )}

      {step === "qris" && placed && (
        <div key="qris" className="auth-rise space-y-4 p-4">
          {!qrisPaid && (
            <button
              onClick={() => setStep("review")}
              className="flex items-center gap-1 rounded-md text-sm text-muted transition-colors hover:text-text focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
            >
              <ArrowLeft className="h-4 w-4" /> Ganti cara pembayaran
            </button>
          )}

          <QrisPaymentPanel
            total={placed.order.total}
            qrValue={placed.qrString || `elkasir:order:${placed.order.id}`}
            qrImageUrl={placed.qrImageUrl}
            status={qrisPaid ? "paid" : "waiting"}
            simulated={placed.simulated}
            onSimulatePaid={simulatePaid}
          />

          {!qrisPaid && (
            <OrderBreakdown
              subtotal={placed.order.subtotal}
              serviceLine={placed.order.serviceLine}
              tax={placed.order.tax}
              total={placed.order.total}
              servicePercent={menu.servicePercent}
              taxPercent={menu.taxEnabled ? menu.taxPercent : undefined}
            />
          )}

          {qrisPaid && (
            <>
              <OrderSummary />
              <p className="text-center text-xs text-muted">
                Mohon tunggu, pesanan Anda sedang disiapkan.
              </p>
              <Button variant="outline" className="w-full" onClick={reset}>
                Pesan lagi
              </Button>
            </>
          )}
        </div>
      )}

      {step === "cashier" && placed && (
        <div key="cashier" className="auth-rise space-y-4 p-4">
          <CashierBarcodePanel
            claimCode={placed.claimCode || placed.order.claimCode || ""}
            total={placed.order.total}
            tableName={placed.order.tableName || menu.table.name}
          />
          <OrderBreakdown
            subtotal={placed.order.subtotal}
            serviceLine={placed.order.serviceLine}
            tax={placed.order.tax}
            total={placed.order.total}
            servicePercent={menu.servicePercent}
            taxPercent={menu.taxEnabled ? menu.taxPercent : undefined}
          />
          <OrderSummary />
          <p className="text-center text-xs text-muted">
            Pesanan tercatat sebagai <span className="font-medium">belum dibayar</span>. Stok baru
            dipotong setelah kasir menerima pembayaran.
          </p>
          <Button variant="outline" className="w-full" onClick={reset}>
            Pesan lagi
          </Button>
        </div>
      )}
    </div>
  );
}
