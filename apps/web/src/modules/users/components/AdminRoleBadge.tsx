import { Badge } from "@/shared/components/ui/badge";
import type { AdminRole, ActiveStatus } from "@/modules/users/types/user.types";

const roleLabel: Record<AdminRole, string> = {
  owner: "Pemilik",
  admin: "Admin",
  manager: "Manajer",
  viewer: "Viewer",
};

const roleTone: Record<AdminRole, "primary" | "neutral" | "warning"> = {
  owner: "primary",
  admin: "primary",
  manager: "warning",
  viewer: "neutral",
};

// Module-owned badge: maps a domain admin role to a generic badge tone.
export function AdminRoleBadge({ role }: { role: AdminRole }) {
  return <Badge tone={roleTone[role]}>{roleLabel[role]}</Badge>;
}

// Module-owned badge: maps a domain active status to a generic badge tone.
export function AdminStatusBadge({ status }: { status: ActiveStatus }) {
  return status === "active" ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>;
}
