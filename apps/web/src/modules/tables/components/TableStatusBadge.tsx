import { Badge } from "@/shared/components/ui/badge";
import type { TableStatus } from "@/modules/tables/types/table.types";

// Module-owned badge: maps a domain table status to a generic badge tone.
export function TableStatusBadge({ status }: { status: TableStatus }) {
  return status === "active" ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>;
}
