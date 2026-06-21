import { Badge, type BadgeProps } from "@/shared/components/ui/badge";

type Tone = NonNullable<BadgeProps["tone"]>;

// Lookup case-insensitive; status penarikan dapat berupa huruf kecil atau Kapital.
const TONE: Record<string, Tone> = {
  success: "success",
  completed: "success",
  pending: "warning",
  unpaid: "warning",
  processing: "primary",
  failed: "danger",
  cancelled: "danger",
  expired: "neutral",
};

const LABEL: Record<string, string> = {
  success: "Berhasil",
  completed: "Selesai",
  pending: "Menunggu",
  unpaid: "Belum bayar",
  processing: "Diproses",
  failed: "Gagal",
  cancelled: "Dibatalkan",
  expired: "Kedaluwarsa",
};

// Module-owned badge: maps a withdrawal status to a generic badge tone.
export function WithdrawalStatusBadge({ status }: { status: string }) {
  const key = status.toLowerCase();
  return <Badge tone={TONE[key] ?? "neutral"}>{LABEL[key] ?? status}</Badge>;
}
