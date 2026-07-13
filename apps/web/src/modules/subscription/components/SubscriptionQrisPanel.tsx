import { QRCodeSVG } from "qrcode.react";
import { Button } from "@/shared/components/ui/button";
import { Card, CardContent } from "@/shared/components/ui/card";

const formatRupiah = (n: number) => `Rp ${n.toLocaleString("id-ID")}`;

// QRIS payment panel for a subscription checkout — mirrors self-order's QrisPaymentPanel
// pattern (img fallback to a client-rendered QR), scoped to this module since it isn't
// domain-agnostic enough for shared/components (invoice-specific copy).
export function SubscriptionQrisPanel({
  amount,
  qrString,
  qrImageUrl,
  checking,
  onCheckStatus,
}: {
  amount: number;
  qrString?: string;
  qrImageUrl?: string;
  checking: boolean;
  onCheckStatus: () => void;
}) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-4 p-6 text-center">
        <p className="text-sm text-muted">Total tagihan</p>
        <p className="text-2xl font-bold tabular-nums text-text">{formatRupiah(amount)}</p>

        <div className="rounded-lg bg-white p-3">
          {qrImageUrl ? (
            <img src={qrImageUrl} alt="Kode QRIS pembayaran" width={190} height={190} />
          ) : (
            <QRCodeSVG value={qrString || "elkasir:subscription"} size={190} marginSize={4} />
          )}
        </div>

        <p className="max-w-xs text-xs text-muted">
          Pindai kode QRIS di atas untuk membayar. Setelah membayar, tekan tombol di bawah untuk
          memeriksa status — tidak ada pembaruan otomatis.
        </p>

        <Button onClick={onCheckStatus} loading={checking} className="w-full">
          Cek Status Pembayaran
        </Button>
      </CardContent>
    </Card>
  );
}
