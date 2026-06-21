import { Badge } from "@/shared/components/ui/badge";

// Module-owned badge: maps a domain transaction status to a generic badge tone.
const TONE: Record<
  string,
  { tone: "neutral" | "primary" | "success" | "warning" | "danger"; label: string }
> = {
  completed: { tone: "success", label: "Selesai" },
  pending: { tone: "warning", label: "Menunggu" },
  voided: { tone: "danger", label: "Dibatalkan" },
  refunded: { tone: "danger", label: "Dikembalikan" },
};

export function TransactionStatusBadge({ status }: { status: string }) {
  const cfg = TONE[status] ?? { tone: "neutral" as const, label: status };
  return <Badge tone={cfg.tone}>{cfg.label}</Badge>;
}
