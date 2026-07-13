import { Badge, type BadgeProps } from "@/shared/components/ui/badge";
import type { WithdrawalStatus } from "@/modules/platform/types/platform.types";

// Exactly the 4 real statuses (§2.7) — no extras, unlike the tenant-side badge this mirrors
// (see PLAN.md §1a/Phase F5, which reconciles that one separately).
const TONE: Record<WithdrawalStatus, BadgeProps["tone"]> = {
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

export function PlatformWithdrawalStatusBadge({ status }: { status: WithdrawalStatus }) {
  return <Badge tone={TONE[status]}>{LABEL[status]}</Badge>;
}
