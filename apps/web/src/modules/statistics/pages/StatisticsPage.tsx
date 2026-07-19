import { useMemo, useState } from "react";
import {
  AreaChart,
  Area,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
  ReferenceLine,
  LabelList,
} from "recharts";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/shared/components/ui/card";
import { ChartTooltip, ChartState, LegendRow } from "@/shared/components/ui/chart";
import { formatIDR, formatNumber } from "@/shared/lib/formatter";
import { formatCompactIDR, formatDayShort, formatMonthShort, rankShade } from "@/shared/lib/chart";
import { useAsync } from "@/shared/hooks/useAsync";
import { cn } from "@/shared/lib/cn";
import { colors, chartPalette } from "@/theme";
import { statisticsService } from "@/modules/statistics/services/statistics.service";

const RANGE_OPTIONS = [
  { value: "7", label: "7 hari" },
  { value: "30", label: "30 hari" },
  { value: "90", label: "90 hari" },
  { value: "12m", label: "Bulanan" },
];

// Money = primary blue; counts = slate. Mono ticks read as data, not prose.
const AXIS_TICK = { fill: colors.muted, fontSize: 11, fontFamily: "var(--font-mono)" } as const;
const PAYMENT_COLORS: Record<string, string> = { Tunai: chartPalette[0], QRIS: chartPalette[3] };

// Local calendar date, not UTC — toISOString() would shift the date back a day for any
// positive UTC-offset timezone (e.g. WIB) when `d` is midnight-local.
function ymd(d: Date) {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}
function daysAgo(n: number) {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d;
}
// First day of the month `n` months back (so "12m" covers this month + 11 prior).
function monthsAgo(n: number) {
  const d = new Date();
  return new Date(d.getFullYear(), d.getMonth() - n, 1);
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="font-mono text-[11px] uppercase tracking-wider text-muted">{label}</p>
      <p className="mt-0.5 font-display text-lg font-semibold tabular-nums text-text">{value}</p>
    </div>
  );
}

export default function StatisticsPage() {
  const [range, setRange] = useState("30");
  const isMonthly = range === "12m";
  const days = Number(range) || 30;

  const period = useMemo(
    () =>
      isMonthly
        ? { from: ymd(monthsAgo(11)), to: ymd(new Date()) }
        : { from: ymd(daysAgo(days)), to: ymd(new Date()) },
    [isMonthly, days],
  );

  const bucketFormatter = isMonthly ? formatMonthShort : formatDayShort;

  const salesQuery = useAsync(async () => {
    if (isMonthly) {
      const rows = await statisticsService.salesByMonth(period);
      return rows.map((r) => ({ bucket: r.month, revenue: r.revenue, txCount: r.txCount }));
    }
    const rows = await statisticsService.sales(period);
    return rows.map((r) => ({ bucket: r.day, revenue: r.revenue, txCount: r.txCount }));
  }, [period.from, period.to, isMonthly]);
  const paymentQuery = useAsync(() => statisticsService.paymentDistribution(), []);
  const categoryQuery = useAsync(
    () => statisticsService.salesByCategory(period),
    [period.from, period.to],
  );
  const topProductsQuery = useAsync(
    () => statisticsService.topProducts({ ...period, limit: 8 }),
    [period.from, period.to],
  );
  const staffQuery = useAsync(
    () => statisticsService.staffPerformance(period),
    [period.from, period.to],
  );

  const sales = salesQuery.data ?? [];
  const payment = paymentQuery.data;
  const categories = categoryQuery.data ?? [];
  const top = topProductsQuery.data ?? [];
  const staff = staffQuery.data ?? [];

  const salesSummary = useMemo(() => {
    if (!sales.length) return null;
    const total = sales.reduce((s, d) => s + d.revenue, 0);
    const tx = sales.reduce((s, d) => s + d.txCount, 0);
    const peak = sales.reduce((m, d) => (d.revenue > m.revenue ? d : m), sales[0]);
    return { total, tx, avg: total / sales.length, peak };
  }, [sales]);

  const paymentData = useMemo(() => {
    if (!payment) return [];
    return [
      { name: "Tunai", value: payment.cashTotal },
      { name: "QRIS", value: payment.qrisTotal },
    ].filter((r) => r.value > 0);
  }, [payment]);
  const paymentTotal = paymentData.reduce((s, r) => s + r.value, 0);

  const categoriesRanked = useMemo(
    () => [...categories].sort((a, b) => b.revenue - a.revenue),
    [categories],
  );
  const staffRanked = useMemo(() => [...staff].sort((a, b) => b.revenue - a.revenue), [staff]);
  const staffMax = Math.max(1, ...staffRanked.map((s) => s.revenue));

  return (
    <div className="space-y-5 p-4 md:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="font-display text-xl font-semibold tracking-tight text-text">Statistik</h2>
          <p className="text-sm text-muted">Analitik mendalam untuk seluruh bisnis Anda.</p>
        </div>
        <div className="inline-flex rounded-lg border border-border bg-surface-muted p-0.5">
          {RANGE_OPTIONS.map((o) => (
            <button
              key={o.value}
              type="button"
              onClick={() => setRange(o.value)}
              className={cn(
                "rounded-md px-3 py-1.5 text-xs font-medium transition-colors",
                range === o.value ? "bg-surface text-text shadow-sm" : "text-muted hover:text-text",
              )}
            >
              {o.label}
            </button>
          ))}
        </div>
      </div>

      {/* Hero — revenue trend with period context (total / avg-per-bucket / best bucket). */}
      <Card>
        <CardHeader className="pb-0">
          <CardTitle className="font-display">Pendapatan</CardTitle>
          <CardDescription>
            {isMonthly ? "Pendapatan bulanan, 12 bulan terakhir" : "Pendapatan harian sepanjang periode"}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 pt-4">
          {salesSummary && (
            <div className="flex flex-wrap gap-x-10 gap-y-3 border-b border-border pb-4">
              <Metric label="Total" value={formatIDR(salesSummary.total)} />
              <Metric
                label={isMonthly ? "Rata-rata / bulan" : "Rata-rata / hari"}
                value={formatIDR(Math.round(salesSummary.avg))}
              />
              <Metric label="Transaksi" value={formatNumber(salesSummary.tx)} />
              <Metric
                label={isMonthly ? "Bulan terbaik" : "Hari terbaik"}
                value={`${bucketFormatter(salesSummary.peak.bucket)} · ${formatCompactIDR(salesSummary.peak.revenue)}`}
              />
            </div>
          )}
          <div className="h-72 md:h-80">
            <ChartState
              loading={salesQuery.loading}
              error={salesQuery.error}
              empty={sales.length === 0}
            >
              <ResponsiveContainer>
                <AreaChart data={sales} margin={{ top: 8, right: 12, left: 4, bottom: 0 }}>
                  <defs>
                    <linearGradient id="revFill" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={colors.primary} stopOpacity={0.28} />
                      <stop offset="100%" stopColor={colors.primary} stopOpacity={0.02} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid vertical={false} stroke={colors.border} strokeOpacity={0.7} />
                  <XAxis
                    dataKey="bucket"
                    tickFormatter={bucketFormatter}
                    tickLine={false}
                    axisLine={false}
                    tickMargin={10}
                    minTickGap={28}
                    tick={AXIS_TICK}
                  />
                  <YAxis
                    tickFormatter={formatCompactIDR}
                    tickLine={false}
                    axisLine={false}
                    width={64}
                    tick={AXIS_TICK}
                  />
                  {salesSummary && (
                    <ReferenceLine
                      y={salesSummary.avg}
                      stroke={colors.muted}
                      strokeDasharray="4 4"
                      strokeOpacity={0.7}
                      label={{
                        value: "rata-rata",
                        position: "insideTopRight",
                        fill: colors.muted,
                        fontSize: 10,
                      }}
                    />
                  )}
                  <Tooltip
                    cursor={{ stroke: colors.primary, strokeOpacity: 0.25, strokeWidth: 1.5 }}
                    content={
                      <ChartTooltip
                        labelFormatter={(l) => bucketFormatter(String(l))}
                        formatter={(v) => formatIDR(v)}
                      />
                    }
                  />
                  <Area
                    type="monotone"
                    dataKey="revenue"
                    name="Pendapatan"
                    stroke={colors.primary}
                    strokeWidth={2.25}
                    fill="url(#revFill)"
                    dot={false}
                    activeDot={{
                      r: 4,
                      strokeWidth: 2,
                      stroke: colors.surface,
                      fill: colors.primary,
                    }}
                  />
                </AreaChart>
              </ResponsiveContainer>
            </ChartState>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="font-display">Volume Transaksi</CardTitle>
            <CardDescription>
              {isMonthly ? "Jumlah transaksi bulanan" : "Jumlah transaksi harian"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-64">
              <ChartState
                loading={salesQuery.loading}
                error={salesQuery.error}
                empty={sales.length === 0}
              >
                <ResponsiveContainer>
                  <BarChart data={sales} margin={{ top: 8, right: 8, left: 4, bottom: 0 }}>
                    <CartesianGrid vertical={false} stroke={colors.border} strokeOpacity={0.7} />
                    <XAxis
                      dataKey="bucket"
                      tickFormatter={bucketFormatter}
                      tickLine={false}
                      axisLine={false}
                      tickMargin={10}
                      minTickGap={28}
                      tick={AXIS_TICK}
                    />
                    <YAxis
                      tickLine={false}
                      axisLine={false}
                      allowDecimals={false}
                      width={32}
                      tick={AXIS_TICK}
                    />
                    <Tooltip
                      cursor={{ fill: colors.surfaceMuted, fillOpacity: 0.6 }}
                      content={
                        <ChartTooltip
                          labelFormatter={(l) => bucketFormatter(String(l))}
                          formatter={(v) => formatNumber(v)}
                        />
                      }
                    />
                    <Bar
                      dataKey="txCount"
                      name="Transaksi"
                      fill={colors.secondary}
                      radius={[4, 4, 0, 0]}
                      maxBarSize={36}
                    />
                  </BarChart>
                </ResponsiveContainer>
              </ChartState>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="font-display">Distribusi Pembayaran</CardTitle>
            <CardDescription>Proporsi pendapatan per metode</CardDescription>
          </CardHeader>
          <CardContent>
            <ChartState
              loading={paymentQuery.loading}
              error={paymentQuery.error}
              empty={paymentData.length === 0}
            >
              <div className="relative h-44">
                <ResponsiveContainer>
                  <PieChart>
                    <Pie
                      data={paymentData}
                      dataKey="value"
                      nameKey="name"
                      innerRadius="62%"
                      outerRadius="92%"
                      paddingAngle={2}
                      stroke="none"
                    >
                      {paymentData.map((e) => (
                        <Cell key={e.name} fill={PAYMENT_COLORS[e.name]} />
                      ))}
                    </Pie>
                    <Tooltip content={<ChartTooltip formatter={(v) => formatIDR(v)} />} />
                  </PieChart>
                </ResponsiveContainer>
                <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
                  <span className="font-mono text-[10px] uppercase tracking-wider text-muted">
                    Total
                  </span>
                  <span className="font-display text-lg font-semibold tabular-nums text-text">
                    {formatCompactIDR(paymentTotal)}
                  </span>
                </div>
              </div>
              <div className="mt-3 space-y-2">
                {paymentData.map((p) => (
                  <LegendRow
                    key={p.name}
                    color={PAYMENT_COLORS[p.name]}
                    label={p.name}
                    value={formatIDR(p.value)}
                    share={paymentTotal > 0 ? Math.round((p.value / paymentTotal) * 100) : 0}
                  />
                ))}
              </div>
            </ChartState>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="font-display">Penjualan per Kategori</CardTitle>
            <CardDescription>Pendapatan menurut kategori menu</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-64">
              <ChartState
                loading={categoryQuery.loading}
                error={categoryQuery.error}
                empty={categoriesRanked.length === 0}
              >
                <ResponsiveContainer>
                  <BarChart
                    data={categoriesRanked}
                    margin={{ top: 8, right: 8, left: 4, bottom: 0 }}
                  >
                    <CartesianGrid vertical={false} stroke={colors.border} strokeOpacity={0.7} />
                    <XAxis
                      dataKey="category"
                      tickLine={false}
                      axisLine={false}
                      tickMargin={10}
                      tick={AXIS_TICK}
                    />
                    <YAxis
                      tickFormatter={formatCompactIDR}
                      tickLine={false}
                      axisLine={false}
                      width={64}
                      tick={AXIS_TICK}
                    />
                    <Tooltip
                      cursor={{ fill: colors.surfaceMuted, fillOpacity: 0.6 }}
                      content={<ChartTooltip formatter={(v) => formatIDR(v)} />}
                    />
                    <Bar dataKey="revenue" name="Penjualan" radius={[5, 5, 0, 0]} maxBarSize={56}>
                      {categoriesRanked.map((c, i) => (
                        <Cell
                          key={c.category}
                          fill={colors.primary}
                          fillOpacity={rankShade(i, categoriesRanked.length)}
                        />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </ChartState>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="font-display">Produk Terlaris</CardTitle>
            <CardDescription>Unit terjual</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-64">
              <ChartState
                loading={topProductsQuery.loading}
                error={topProductsQuery.error}
                empty={top.length === 0}
              >
                <ResponsiveContainer>
                  <BarChart data={top} layout="vertical" margin={{ left: 8, right: 28 }}>
                    <CartesianGrid horizontal={false} stroke={colors.border} strokeOpacity={0.7} />
                    <XAxis
                      type="number"
                      tickLine={false}
                      axisLine={false}
                      allowDecimals={false}
                      tick={AXIS_TICK}
                      hide
                    />
                    <YAxis
                      dataKey="productName"
                      type="category"
                      tickLine={false}
                      axisLine={false}
                      width={120}
                      tick={{ ...AXIS_TICK, fontFamily: undefined }}
                    />
                    <Tooltip
                      cursor={{ fill: colors.surfaceMuted, fillOpacity: 0.6 }}
                      content={<ChartTooltip formatter={(v) => formatNumber(v)} />}
                    />
                    <Bar dataKey="qty" name="Terjual" radius={[0, 4, 4, 0]} maxBarSize={22}>
                      {top.map((p, i) => (
                        <Cell
                          key={p.productName}
                          fill={colors.primary}
                          fillOpacity={rankShade(i, top.length)}
                        />
                      ))}
                      <LabelList
                        dataKey="qty"
                        position="right"
                        fill={colors.muted}
                        fontSize={11}
                        className="font-mono"
                      />
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </ChartState>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="font-display">Kinerja Staf</CardTitle>
          <CardDescription>Peringkat pendapatan per staf</CardDescription>
        </CardHeader>
        <CardContent>
          <ChartState
            loading={staffQuery.loading}
            error={staffQuery.error}
            empty={staffRanked.length === 0}
          >
            <div className="space-y-4">
              {staffRanked.map((c, i) => (
                <div key={c.staffId}>
                  <div className="mb-1.5 flex items-center justify-between gap-3 text-sm">
                    <div className="flex items-center gap-2.5">
                      <span
                        className={cn(
                          "flex h-6 w-6 items-center justify-center rounded-full font-mono text-xs font-semibold tabular-nums",
                          i === 0
                            ? "bg-primary text-primary-foreground"
                            : "bg-surface-muted text-muted",
                        )}
                      >
                        {i + 1}
                      </span>
                      <span className="font-medium text-text">{c.name}</span>
                    </div>
                    <div className="text-right">
                      <div className="font-mono font-medium tabular-nums text-text">
                        {formatIDR(c.revenue)}
                      </div>
                      <div className="font-mono text-xs tabular-nums text-muted">
                        {formatNumber(c.txCount)} transaksi
                      </div>
                    </div>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-surface-muted">
                    <div
                      className="h-full rounded-full bg-primary"
                      style={{
                        width: `${(c.revenue / staffMax) * 100}%`,
                        opacity: rankShade(i, staffRanked.length),
                      }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </ChartState>
        </CardContent>
      </Card>
    </div>
  );
}
