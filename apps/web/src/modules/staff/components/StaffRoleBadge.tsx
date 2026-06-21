import { Badge } from "@/shared/components/ui/badge";
import type { StaffRole, ActiveStatus } from "@/modules/staff/types/staff.types";

const roleLabel: Record<StaffRole, string> = { cashier: "Kasir", supervisor: "Supervisor" };

// Module-owned badge: maps a domain staff role to a generic badge tone.
export function StaffRoleBadge({ role }: { role: StaffRole }) {
  return <Badge tone={role === "supervisor" ? "primary" : "neutral"}>{roleLabel[role]}</Badge>;
}

// Module-owned badge: maps a domain active status to a generic badge tone.
export function StaffStatusBadge({ status }: { status: ActiveStatus }) {
  return status === "active" ? (
    <Badge tone="success">Aktif</Badge>
  ) : (
    <Badge tone="neutral">Nonaktif</Badge>
  );
}
