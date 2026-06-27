import { Badge, type BadgeProps } from "@/shared/components/ui/badge";
import type { CashMovementType } from "@/modules/cash-movements/types/cash-movement.types";

const LABEL: Record<CashMovementType, string> = {
  capital: "Modal Tambahan",
  expense: "Biaya Operasional",
  adjustment: "Penyesuaian Kas",
};

const TONE: Record<CashMovementType, NonNullable<BadgeProps["tone"]>> = {
  capital: "success",
  expense: "warning",
  adjustment: "neutral",
};

// Module-owned badge: maps a cash-movement type to a generic badge tone.
export function CashMovementTypeBadge({ type }: { type: CashMovementType }) {
  return <Badge tone={TONE[type] ?? "neutral"}>{LABEL[type] ?? type}</Badge>;
}
