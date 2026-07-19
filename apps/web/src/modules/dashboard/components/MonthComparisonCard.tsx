import { ArrowDownRight, ArrowUpRight, Minus } from "lucide-react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/shared/components/ui/card";
import { ChartState } from "@/shared/components/ui/chart";
import { formatIDR } from "@/shared/lib/formatter";
import { cn } from "@/shared/lib/cn";

export function MonthComparisonCard({
  loading,
  error,
  thisMonthLabel,
  lastMonthLabel,
  thisRevenue,
  lastRevenue,
}: {
  loading: boolean;
  error: string | null;
  thisMonthLabel: string;
  lastMonthLabel: string;
  thisRevenue: number;
  lastRevenue: number;
}) {
  // No baseline (e.g. brand-new store) → nothing meaningful to compare against.
  const delta = lastRevenue > 0 ? ((thisRevenue - lastRevenue) / lastRevenue) * 100 : null;
  const trend = delta == null || Math.abs(delta) < 0.05 ? "flat" : delta > 0 ? "up" : "down";

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="font-display">Perbandingan Bulanan</CardTitle>
        <CardDescription>
          Pendapatan {thisMonthLabel} (s/d hari ini) vs periode yang sama di {lastMonthLabel}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ChartState loading={loading} error={error} empty={false}>
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <p className="font-mono text-[11px] uppercase tracking-wider text-muted">
                {thisMonthLabel}
              </p>
              <p className="mt-1 font-display text-2xl font-semibold tabular-nums text-text">
                {formatIDR(thisRevenue)}
              </p>
            </div>

            <div
              className={cn(
                "flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium",
                trend === "up" && "bg-success-soft text-success",
                trend === "down" && "bg-danger-soft text-danger",
                trend === "flat" && "bg-surface-muted text-muted",
              )}
            >
              {trend === "up" && <ArrowUpRight className="h-3.5 w-3.5" />}
              {trend === "down" && <ArrowDownRight className="h-3.5 w-3.5" />}
              {trend === "flat" && <Minus className="h-3.5 w-3.5" />}
              {delta == null ? "Belum ada pembanding" : `${Math.abs(delta).toFixed(1)}%`}
            </div>

            <div className="text-right">
              <p className="font-mono text-[11px] uppercase tracking-wider text-muted">
                {lastMonthLabel}
              </p>
              <p className="mt-1 font-display text-lg font-medium tabular-nums text-muted">
                {formatIDR(lastRevenue)}
              </p>
            </div>
          </div>
        </ChartState>
      </CardContent>
    </Card>
  );
}
