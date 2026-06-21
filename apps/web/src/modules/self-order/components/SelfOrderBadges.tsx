import { Badge, type BadgeProps } from "@/shared/components/ui/badge";
import {
  ORDER_STAGE_LABEL,
  type SelfOrderPaymentStatus,
  type SelfOrderStatus,
} from "@/modules/self-order/types/self-order.types";

type Tone = BadgeProps["tone"];

// Module-owned badges: map self-order domain statuses to a generic badge tone +
// Indonesian label. The shared <Badge> stays domain-agnostic.

const ORDER_STAGE_TONE: Record<SelfOrderStatus, Tone> = {
  placed: "primary",
  preparing: "warning",
  completed: "success",
};

export function OrderStageBadge({ status }: { status: SelfOrderStatus }) {
  return <Badge tone={ORDER_STAGE_TONE[status]}>{ORDER_STAGE_LABEL[status]}</Badge>;
}

const PAYMENT_STATUS: Record<SelfOrderPaymentStatus, { label: string; tone: Tone }> = {
  paid: { label: "Lunas", tone: "success" },
  pending: { label: "Menunggu", tone: "warning" },
  unpaid: { label: "Belum bayar", tone: "warning" },
  expired: { label: "Kedaluwarsa", tone: "neutral" },
  failed: { label: "Gagal", tone: "danger" },
};

export function PaymentStatusBadge({ status }: { status: SelfOrderPaymentStatus }) {
  const s = PAYMENT_STATUS[status] ?? { label: status, tone: "neutral" as Tone };
  return <Badge tone={s.tone}>{s.label}</Badge>;
}
