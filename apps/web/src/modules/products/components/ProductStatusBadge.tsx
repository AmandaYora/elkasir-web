import { Badge } from "@/shared/components/ui/badge";
import type { ProductStatus } from "@/modules/products/types/product.types";

// Module-owned badge: maps a domain product status to a generic badge tone.
export function ProductStatusBadge({ status }: { status: ProductStatus }) {
  return status === "active" ? (
    <Badge tone="success">Aktif</Badge>
  ) : (
    <Badge tone="neutral">Nonaktif</Badge>
  );
}
