import { Badge } from "@/shared/components/ui/badge";
import type { PlatformUserStatus } from "@/modules/platform/types/platform.types";

export function PlatformUserStatusBadge({ status }: { status: PlatformUserStatus }) {
  return (
    <Badge tone={status === "active" ? "success" : "neutral"}>
      {status === "active" ? "Aktif" : "Nonaktif"}
    </Badge>
  );
}
