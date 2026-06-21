import { Badge } from "@/shared/components/ui/badge";
import type { ShiftStatus } from "@/modules/shifts/types/shift.types";

// Module-owned badge: maps a domain shift status to a generic badge tone.
export function ShiftStatusBadge({ status }: { status: ShiftStatus }) {
  return status === "open" ? (
    <Badge tone="success">Aktif</Badge>
  ) : (
    <Badge tone="neutral">Selesai</Badge>
  );
}
