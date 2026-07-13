import { Badge, type BadgeProps } from "@/shared/components/ui/badge";
import type { WithdrawalStatus } from "@/modules/withdrawals/types/withdrawal.types";

// Exactly the 4 real statuses (§2.7/§1a) — this component previously mapped 8 keys
// (success/completed/pending/unpaid/processing/failed/cancelled/expired), which looked
// copy-pasted from the self-order payment badge and never matched what this endpoint actually
// returns. Reconciled to the real claim -> complete vocabulary only.
const TONE: Record<WithdrawalStatus, NonNullable<BadgeProps["tone"]>> = {
  pending: "warning",
  processing: "primary",
  success: "success",
  failed: "danger",
};

const LABEL: Record<WithdrawalStatus, string> = {
  pending: "Menunggu",
  processing: "Sedang diproses",
  success: "Berhasil",
  failed: "Ditolak",
};

export function WithdrawalStatusBadge({ status }: { status: WithdrawalStatus }) {
  return <Badge tone={TONE[status]}>{LABEL[status]}</Badge>;
}
