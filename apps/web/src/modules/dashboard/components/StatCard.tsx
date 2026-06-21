import type { LucideIcon } from "lucide-react";
import { Card, CardContent } from "@/shared/components/ui/card";

type Accent = "primary" | "info" | "success" | "warning";

// Themed accent classes (no shadcn tokens) for the icon chip.
const accentClasses: Record<Accent, string> = {
  primary: "bg-primary-soft text-primary",
  info: "bg-primary-soft text-primary",
  success: "bg-success-soft text-success",
  warning: "bg-warning-soft text-warning",
};

export function StatCard({
  label,
  value,
  icon: Icon,
  accent = "primary",
}: {
  label: string;
  value: string;
  icon: LucideIcon;
  accent?: Accent;
}) {
  return (
    <Card>
      <CardContent className="flex items-center gap-4 p-4">
        <div
          className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-lg ${accentClasses[accent]}`}
        >
          <Icon className="h-5 w-5" />
        </div>
        <div className="min-w-0">
          <p className="text-xs font-medium uppercase tracking-wider text-muted">{label}</p>
          <p className="mt-0.5 truncate text-xl font-semibold text-text tabular-nums">{value}</p>
        </div>
      </CardContent>
    </Card>
  );
}
