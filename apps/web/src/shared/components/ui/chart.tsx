// Shared chart primitives so every chart across dashboard + statistics speaks one visual
// language: the same tooltip, the same loading/empty/error states. Generic, domain-agnostic.
import { cn } from "@/shared/lib/cn";

export type TooltipEntry = {
  color?: string;
  fill?: string;
  name?: string;
  value: number;
  dataKey?: string | number;
};

// Recharts injects active/payload/label; the page supplies formatter/labelFormatter.
export function ChartTooltip({
  active,
  payload,
  label,
  formatter,
  labelFormatter,
}: {
  active?: boolean;
  payload?: TooltipEntry[];
  label?: string | number;
  formatter?: (value: number, name?: string) => string;
  labelFormatter?: (label: string | number) => string;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div className="min-w-36 rounded-md border border-border bg-surface/95 p-2.5 shadow-lg backdrop-blur-sm">
      {label !== undefined && label !== "" && (
        <div className="mb-1.5 border-b border-border/60 pb-1.5 font-display text-xs font-semibold text-text">
          {labelFormatter ? labelFormatter(label) : label}
        </div>
      )}
      <div className="space-y-1">
        {payload.map((p, i) => (
          <div key={i} className="flex items-center gap-2 text-xs">
            <span
              className="h-2 w-2 shrink-0 rounded-full"
              style={{ background: p.color ?? p.fill }}
            />
            <span className="text-muted">{p.name}</span>
            <span className="ml-auto pl-3 font-mono font-medium tabular-nums text-text">
              {formatter ? formatter(p.value, p.name) : p.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// Fills its parent box (which sets the height). Loading shows a pulse skeleton, not a bare spinner.
export function ChartState({
  loading,
  error,
  empty,
  children,
}: {
  loading: boolean;
  error: string | null;
  empty: boolean;
  children: React.ReactNode;
}) {
  if (loading) return <div className="h-full w-full animate-pulse rounded-md bg-surface-muted" />;
  if (error)
    return (
      <div className="flex h-full w-full items-center justify-center rounded-md bg-danger-soft/40 px-4 text-center text-sm text-danger">
        Gagal memuat data.
      </div>
    );
  if (empty)
    return (
      <div className="flex h-full w-full items-center justify-center text-sm text-muted">
        Belum ada data
      </div>
    );
  return <>{children}</>;
}

// Small legend row: color dot · label · right-aligned value (+ optional share %). Used under donuts.
export function LegendRow({
  color,
  label,
  value,
  share,
}: {
  color: string;
  label: string;
  value: string;
  share?: number;
}) {
  return (
    <div className="flex items-center justify-between gap-2 text-sm">
      <div className="flex items-center gap-2">
        <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ background: color }} />
        <span className="text-text">{label}</span>
      </div>
      <div className="flex items-baseline gap-1.5">
        <span className="font-mono font-medium tabular-nums text-text">{value}</span>
        {share !== undefined && (
          <span className={cn("font-mono text-xs tabular-nums text-muted")}>{share}%</span>
        )}
      </div>
    </div>
  );
}
