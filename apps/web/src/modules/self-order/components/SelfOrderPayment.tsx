import type { ComponentType } from "react";
import { QRCodeSVG } from "qrcode.react";
import BarcodeImpl from "react-barcode";
import { Loader2, CheckCircle2, ScanLine } from "lucide-react";

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

// QrisPaymentPanel → pembayaran QRIS (QR dari gateway, status dari polling).
// CashierBarcodePanel → bayar di kasir (kode klaim ditampilkan sebagai barcode Code128
// untuk dipindai scanner kasir; kode juga ditampilkan untuk diketik manual).

export function QrisPaymentPanel({
  total,
  qrValue,
  status,
  simulated,
  onSimulatePaid,
}: {
  total: number;
  qrValue: string;
  status: "waiting" | "paid";
  simulated?: boolean;
  onSimulatePaid?: () => void;
}) {
  if (status === "paid") {
    return (
      <div className="flex flex-col items-center gap-3 rounded-xl border border-border bg-surface p-6 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-success-soft text-success">
          <CheckCircle2 className="h-7 w-7" />
        </div>
        <div className="text-base font-semibold">Pembayaran berhasil</div>
        <p className="text-sm text-muted">
          QRIS {formatIDR(total)} diterima. Pesanan Anda diteruskan ke dapur.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center gap-3 rounded-xl border border-border bg-surface p-5">
      <Badge tone="primary">Pembayaran QRIS</Badge>
      <div className="rounded-lg bg-white p-3">
        <QRCodeSVG value={qrValue} size={190} marginSize={4} />
      </div>
      <div className="flex items-center gap-2 text-sm font-medium text-warning">
        <Loader2 className="h-4 w-4 animate-spin" />
        Menunggu pembayaran…
      </div>
      <div className="text-lg font-bold">{formatIDR(total)}</div>
      <p className="text-center text-xs text-muted">
        Scan QR ini dengan aplikasi bank atau e-wallet Anda. Pesanan otomatis masuk setelah
        pembayaran lunas.
      </p>
      {simulated && onSimulatePaid && (
        <Button variant="outline" className="w-full" onClick={onSimulatePaid}>
          Tandai sudah dibayar
        </Button>
      )}
    </div>
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
  return (
    <div className="flex flex-col items-center gap-3 rounded-xl border border-border bg-surface p-5 text-center">
      <Badge tone="primary" className="gap-1.5">
        <ScanLine className="h-3.5 w-3.5" /> Bayar di kasir
      </Badge>
      <div className="rounded-lg bg-white p-3">
        <Barcode
          value={claimCode}
          format="CODE128"
          height={56}
          width={1.6}
          displayValue={false}
          margin={0}
          background="#ffffff"
        />
      </div>
      <div className="font-mono text-sm font-semibold tracking-[0.18em]">{claimCode}</div>
      <div className="text-lg font-bold">{formatIDR(total)}</div>
      <p className="text-center text-sm text-muted">
        Tunjukkan kode ini ke kasir untuk Meja{" "}
        <span className="font-semibold text-text">{tableName}</span>. Kasir memindai atau mengetik
        kode, lalu Anda membayar tunai.
      </p>
    </div>
  );
}
