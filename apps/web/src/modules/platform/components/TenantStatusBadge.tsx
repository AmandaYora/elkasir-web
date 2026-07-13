import { Badge } from "@/shared/components/ui/badge";

export function TenantStatusBadge({ status }: { status: "active" | "suspended" }) {
  return (
    <Badge tone={status === "active" ? "success" : "danger"}>
      {status === "active" ? "Aktif" : "Nonaktif"}
    </Badge>
  );
}
