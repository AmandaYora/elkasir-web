import { useEffect, useRef, useState, type ComponentType } from "react";
import { createPortal } from "react-dom";
import { QRCodeSVG } from "qrcode.react";
import BarcodeImpl from "react-barcode";
import { Loader2, CheckCircle2, ScanLine, Maximize2, X } from "lucide-react";

// react-barcode ships class-based types incompatible with React 19's JSX types;
// re-type it as a functional component with the props we use.
const Barcode = BarcodeImpl as unknown as ComponentType<{
  value: string;
  format?: string;
  height?: number;
  width?: number;
  displayValue?: boolean;
  margin?: number;
  background?: string;
}>;
import { Badge } from "@/shared/components/ui/badge";
import { Button } from "@/shared/components/ui/button";
import { formatIDR } from "@/shared/lib/formatter";
import { useElementWidth } from "@/modules/self-order/hooks/useElementWidth";

// QrisPaymentPanel → pembayaran QRIS (QR dari gateway, status di-push via SSE/callback).
// CashierBarcodePanel → bayar di kasir, disajikan sebagai "tiket" (kode klaim + barcode
// Code128) — metafora tiket/struk asli agar terasa seperti benda yang memang dibawa ke
// kasir, bukan sekadar kartu info. Keduanya berbagi mode "perbesar layar" (ScreenBoost)
// karena device pelanggan sangat variatif — layar kecil/redup tetap bisa dipindai dengan
// membesarkan barcode/QR ke kontras maksimum (putih penuh) tanpa kehilangan ketajaman
// (keduanya di-render sebagai SVG, jadi aman di-skalakan ke ukuran berapa pun).
// OrderBreakdown → rincian biaya: Subtotal, Layanan (service + biaya gateway), PPN, Total.

export function OrderBreakdown({
  subtotal,
  serviceLine,
  tax,
  total,
  servicePercent,
  taxPercent,
}: {
  subtotal: number;
  serviceLine: number;
  tax: number;
  total: number;
  servicePercent?: number;
  taxPercent?: number;
}) {
  const Row = ({ label, value, strong }: { label: string; value: number; strong?: boolean }) => (
    <div
      className={`flex items-center justify-between px-4 py-2.5 text-sm ${strong ? "font-semibold" : ""}`}
    >
      <span className={strong ? "text-text" : "text-muted"}>{label}</span>
      <span className={strong ? "font-mono tabular-nums" : "tabular-nums"}>{formatIDR(value)}</span>
    </div>
  );
  // Sertakan persen pada label agar rincian "menjelaskan dirinya" ke pelanggan awam.
  const serviceLabel = servicePercent ? `Layanan (${servicePercent}%)` : "Layanan";
  const taxLabel = taxPercent ? `PPN (${taxPercent}%)` : "PPN";
  return (
    <div className="w-full rounded-2xl border border-border bg-surface">
      <Row label="Subtotal" value={subtotal} />
      {serviceLine > 0 && <Row label={serviceLabel} value={serviceLine} />}
      {tax > 0 && <Row label={taxLabel} value={tax} />}
      <div className="border-t border-border">
        <Row label="Total" value={total} strong />
      </div>
    </div>
  );
}

// Titik-titik "sobekan" yang menyatukan dua bagian tiket — dua lingkaran kecil berwarna
// senada latar halaman (bukan latar kartu) untuk ilusi lubang perforasi, plus garis putus.
function TicketPerforation() {
  return (
    <div className="relative border-t border-dashed border-border">
      <span className="absolute -left-[9px] top-1/2 h-[18px] w-[18px] -translate-y-1/2 rounded-full bg-surface-muted" />
      <span className="absolute -right-[9px] top-1/2 h-[18px] w-[18px] -translate-y-1/2 rounded-full bg-surface-muted" />
    </div>
  );
}

// Barcode Code128 di-skala presisi ke lebar container yang tersedia lewat CSS transform
// (bukan lewat prop width react-barcode, yang linear terhadap jumlah modul dan sulit
// ditebak dari panjang kode yang variatif per meja). Karena react-barcode merender SVG,
// men-skala naik/turun tetap tajam — aman dipakai juga di mode perbesar layar.
function ScaledBarcode({ value }: { value: string }) {
  const { ref, width: containerWidth } = useElementWidth<HTMLDivElement>();
  const innerRef = useRef<HTMLDivElement>(null);
  const [naturalWidth, setNaturalWidth] = useState(0);
  const baseHeight = 64;

  useEffect(() => {
    setNaturalWidth(innerRef.current?.scrollWidth ?? 0);
  }, [value]);

  const scale =
    naturalWidth > 0 && containerWidth > 0
      ? Math.min(2.2, (containerWidth * 0.92) / naturalWidth)
      : 1;

  return (
    <div
      ref={ref}
      className="flex w-full items-center justify-center overflow-hidden"
      style={{ height: naturalWidth ? baseHeight * scale : baseHeight }}
    >
      <div ref={innerRef} style={{ transform: `scale(${scale})`, transformOrigin: "center" }}>
        <Barcode
          value={value}
          format="CODE128"
          height={baseHeight}
          width={2}
          displayValue={false}
          margin={0}
          background="#ffffff"
        />
      </div>
    </div>
  );
}

// Overlay layar-penuh, kontras maksimum (putih solid) untuk memudahkan pemindaian saat
// layar ponsel kecil/redup — device pelanggan sangat variatif, jadi ini jaring pengaman
// yang sama untuk semua ukuran layar, bukan cuma penyesuaian responsif pasif.
function ScreenBoostOverlay({
  open,
  onClose,
  eyebrow,
  code,
  amount,
  children,
}: {
  open: boolean;
  onClose: () => void;
  eyebrow: string;
  code?: string;
  amount: number;
  children: React.ReactNode;
}) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = "";
    };
  }, [open, onClose]);

  if (!open) return null;

  return createPortal(
    <div
      role="dialog"
      aria-modal="true"
      aria-label={eyebrow}
      onClick={onClose}
      // bg-white sengaja bukan token tema (bg-surface): tujuannya kontras maksimum untuk
      // scanner kasir, harus tetap putih murni terlepas dari tema/skin apa pun ke depannya.
      className="overlay-fade fixed inset-0 z-50 flex flex-col items-center justify-center gap-6 bg-white p-6"
    >
      <button
        onClick={onClose}
        aria-label="Tutup"
        className="absolute right-4 top-4 rounded-full p-2.5 text-muted transition-colors hover:bg-surface-muted hover:text-text focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
      >
        <X className="h-6 w-6" />
      </button>
      <div
        onClick={(e) => e.stopPropagation()}
        className="flex w-full max-w-xs flex-col items-center gap-5"
      >
        <span className="text-xs font-semibold uppercase tracking-[0.2em] text-muted">
          {eyebrow}
        </span>
        {children}
        {code && (
          <div className="font-mono text-xl font-bold tracking-[0.2em] text-text">{code}</div>
        )}
        <div className="font-display text-3xl font-extrabold text-text">{formatIDR(amount)}</div>
      </div>
      <p className="max-w-xs text-center text-sm text-muted">
        Tunjukkan layar ini ke kasir untuk dipindai.
      </p>
    </div>,
    document.body,
  );
}

export function QrisPaymentPanel({
  total,
  qrValue,
  qrImageUrl,
  status,
  simulated,
  onSimulatePaid,
}: {
  total: number;
  qrValue: string;
  // Midtrans QRIS hanya memberi URL gambar QR (bukan string mentah). Bila ada, tampilkan
  // gambar resmi gateway; jika tidak, render QR dari qrValue (fallback string/simulasi).
  qrImageUrl?: string;
  status: "waiting" | "paid";
  simulated?: boolean;
  onSimulatePaid?: () => void;
}) {
  const { ref, width } = useElementWidth<HTMLDivElement>();
  const [boost, setBoost] = useState(false);
  const qrSize = Math.round(Math.min(220, Math.max(140, width * 0.72 || 190)));

  if (status === "paid") {
    return (
      <div className="auth-rise flex flex-col items-center gap-3 rounded-2xl border border-border bg-surface p-6 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-success-soft text-success">
          <CheckCircle2 className="h-7 w-7" />
        </div>
        <div className="font-display text-lg font-bold text-text">Pembayaran berhasil</div>
        <p className="text-sm text-muted">
          QRIS {formatIDR(total)} diterima. Pesanan Anda diteruskan ke dapur.
        </p>
      </div>
    );
  }

  return (
    <>
      <div className="flex flex-col items-center gap-3 rounded-2xl border border-border bg-surface p-5">
        <Badge tone="primary">Pembayaran QRIS</Badge>
        {/* bg-white (bukan token tema) di setiap chip QR/barcode di file ini: kontras
            pemindaian harus tetap putih murni, lepas dari tema warna aplikasi. */}
        <div
          ref={ref}
          className="flex items-center justify-center rounded-xl bg-white p-3 ring-1 ring-border/70"
        >
          {qrImageUrl ? (
            <img
              src={qrImageUrl}
              alt="Kode QRIS pembayaran"
              width={qrSize}
              height={qrSize}
              style={{ width: qrSize, height: qrSize }}
            />
          ) : (
            <QRCodeSVG value={qrValue} size={qrSize} marginSize={4} />
          )}
        </div>
        <div className="flex items-center gap-2 text-sm font-medium text-warning">
          <Loader2 className="h-4 w-4 animate-spin" />
          Menunggu pembayaran…
        </div>
        <div className="font-display text-xl font-extrabold text-text">{formatIDR(total)}</div>
        <p className="text-center text-xs text-muted">
          Scan QR ini dengan aplikasi bank atau e-wallet Anda. Pesanan otomatis masuk setelah
          pembayaran lunas.
        </p>
        <button
          onClick={() => setBoost(true)}
          className="flex items-center gap-1.5 rounded-md text-xs font-semibold text-primary transition-colors hover:text-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
        >
          <Maximize2 className="h-3.5 w-3.5" /> Perbesar tampilan
        </button>
        {simulated && onSimulatePaid && (
          <Button variant="outline" className="w-full" onClick={onSimulatePaid}>
            Tandai sudah dibayar
          </Button>
        )}
      </div>

      <ScreenBoostOverlay
        open={boost}
        onClose={() => setBoost(false)}
        eyebrow="Pembayaran QRIS"
        amount={total}
      >
        <div className="flex w-full items-center justify-center rounded-xl bg-white p-4">
          {qrImageUrl ? (
            <img src={qrImageUrl} alt="Kode QRIS pembayaran" className="h-auto w-full max-w-70" />
          ) : (
            <QRCodeSVG value={qrValue} size={280} marginSize={4} />
          )}
        </div>
      </ScreenBoostOverlay>
    </>
  );
}

export function CashierBarcodePanel({
  claimCode,
  total,
  tableName,
}: {
  claimCode: string;
  total: number;
  tableName: string;
}) {
  const [boost, setBoost] = useState(false);

  return (
    <>
      <div className="overflow-hidden rounded-2xl border border-border bg-surface shadow-sm">
        <div className="flex flex-col items-center gap-3 p-5 text-center sm:p-6">
          <Badge tone="primary" className="gap-1.5">
            <ScanLine className="h-3.5 w-3.5" /> Bayar di kasir
          </Badge>
          <div className="w-full max-w-70 rounded-xl bg-white p-4 ring-1 ring-border/70">
            <ScaledBarcode value={claimCode} />
          </div>
          <div className="font-mono text-base font-bold tracking-[0.18em] text-text sm:text-lg">
            {claimCode}
          </div>
          <div className="font-display text-2xl font-extrabold text-text">{formatIDR(total)}</div>
          <button
            onClick={() => setBoost(true)}
            className="flex items-center gap-1.5 rounded-md text-xs font-semibold text-primary transition-colors hover:text-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
          >
            <Maximize2 className="h-3.5 w-3.5" /> Perbesar tampilan
          </button>
        </div>
        <TicketPerforation />
        <div className="p-5 pt-4 text-center sm:p-6 sm:pt-4">
          <p className="text-sm text-muted">
            Tunjukkan kode ini ke kasir untuk Meja{" "}
            <span className="font-semibold text-text">{tableName}</span>. Kasir memindai atau
            mengetik kode, lalu Anda membayar tunai.
          </p>
        </div>
      </div>

      <ScreenBoostOverlay
        open={boost}
        onClose={() => setBoost(false)}
        eyebrow={`Meja ${tableName}`}
        code={claimCode}
        amount={total}
      >
        <div className="w-full rounded-xl bg-white p-4">
          <ScaledBarcode value={claimCode} />
        </div>
      </ScreenBoostOverlay>
    </>
  );
}
