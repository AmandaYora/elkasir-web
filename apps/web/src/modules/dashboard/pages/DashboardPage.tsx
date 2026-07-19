import { useMemo } from "react";
import {
  DollarSign,
  Receipt,
  Wallet,
  QrCode,
  ShoppingBag,
  Sparkles,
  Landmark,
  type LucideIcon,
} from "lucide-react";
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
import { formatIDR, formatDateTime, formatNumber } from "@/shared/lib/formatter";
import { formatCompactIDR, formatDayShort, rankShade } from "@/shared/lib/chart";
import { useAsync } from "@/shared/hooks/useAsync";
import { useAuthStore } from "@/shared/stores/auth.store";
import { colors, chartPalette } from "@/theme";
import { StatCard } from "@/modules/dashboard/components/StatCard";
import { MonthComparisonCard } from "@/modules/dashboard/components/MonthComparisonCard";
import { dashboardService } from "@/modules/dashboard/services/dashboard.service";

const AXIS_TICK = { fill: colors.muted, fontSize: 11, fontFamily: "var(--font-mono)" } as const;
const PAYMENT_COLORS: Record<string, string> = { Tunai: chartPalette[0], QRIS: chartPalette[3] };

const monthLabelFmt = new Intl.DateTimeFormat("id-ID", { month: "long", year: "numeric" });

// Local calendar date, not UTC — toISOString() would shift the date back a day for any
// positive UTC-offset timezone (e.g. WIB) when `d` is midnight-local (as month boundaries are).
function ymd(d: Date) {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

// Month-to-date vs. the same date range last month (apples-to-apples, not partial-vs-full-month).
function monthCompareRanges() {
  const now = new Date();
  const day = now.getDate();
  const thisStart = new Date(now.getFullYear(), now.getMonth(), 1);
  const thisEnd = new Date(now.getFullYear(), now.getMonth(), day + 1);
  const lastStart = new Date(now.getFullYear(), now.getMonth() - 1, 1);
  const lastEndRaw = new Date(now.getFullYear(), now.getMonth() - 1, day + 1);
  const lastEnd = lastEndRaw > thisStart ? thisStart : lastEndRaw; // cap short months (e.g. Feb)
  return {
    thisMonth: { from: ymd(thisStart), to: ymd(thisEnd) },
    lastMonth: { from: ymd(lastStart), to: ymd(lastEnd) },
    thisLabel: monthLabelFmt.format(thisStart),
    lastLabel: monthLabelFmt.format(lastStart),
  };
}

const sourceLabel: Record<string, string> = {
  cashier: "Kasir",
  self_order: "Pesan Mandiri",
};
const methodLabel: Record<string, string> = {
  cash: "Tunai",
  qris: "QRIS",
};

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user);
  const firstName = user?.name.split(" ")[0] ?? "Admin";

  const dashboardQuery = useAsync(() => dashboardService.dashboard(), []);
  const salesQuery = useAsync(() => dashboardService.sales(), []);
  const paymentQuery = useAsync(() => dashboardService.paymentDistribution(), []);
  const topProductsQuery = useAsync(() => dashboardService.topProducts({ limit: 7 }), []);
  const categoryQuery = useAsync(() => dashboardService.salesByCategory(), []);

  const monthRanges = useMemo(() => monthCompareRanges(), []);
  const thisMonthQuery = useAsync(
    () => dashboardService.dashboard(monthRanges.thisMonth),
    [monthRanges.thisMonth.from, monthRanges.thisMonth.to],
  );
  const lastMonthQuery = useAsync(
    () => dashboardService.dashboard(monthRanges.lastMonth),
    [monthRanges.lastMonth.from, monthRanges.lastMonth.to],
  );

  const summary = dashboardQuery.data?.summary;
  const recent = dashboardQuery.data?.recent ?? [];

  const salesData = (salesQuery.data ?? []).map((d) => ({
    day: d.day,
    revenue: d.revenue,
    txCount: d.txCount,
  }));

  const payment = paymentQuery.data;
  const paymentData = payment
    ? [
        { name: "Tunai", value: payment.cashTotal },
        { name: "QRIS", value: payment.qrisTotal },
      ].filter((r) => r.value > 0)
    : [];
  const paymentTotal = paymentData.reduce((s, r) => s + r.value, 0);

  const topProducts = (topProductsQuery.data ?? []).map((p) => ({
    name: p.productName,
    qty: p.qty,
    revenue: p.revenue,
  }));

  const categories = [...(categoryQuery.data ?? [])].sort((a, b) => b.revenue - a.revenue);
  const categoryMax = Math.max(1, ...categories.map((c) => c.revenue));

  return (
    <div className="space-y-5 p-4 md:p-6">
      <div>
        <h2 className="font-display text-xl font-semibold tracking-tight text-text">
          Selamat datang, {firstName}
        </h2>
        <p className="text-sm text-muted">Ringkasan kinerja toko Anda 30 hari terakhir.</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label="Pendapatan"
          value={formatIDR(summary?.revenue ?? 0)}
          icon={DollarSign}
          accent="primary"
        />
        <StatCard
          label="Transaksi"
          value={formatNumber(summary?.txCount ?? 0)}
          icon={Receipt}
          accent="info"
        />
        <StatCard
          label="Tunai"
          value={formatIDR(summary?.cashTotal ?? 0)}
          icon={Wallet}
          accent="success"
        />
        <StatCard
          label="QRIS"
          value={formatIDR(summary?.qrisTotal ?? 0)}
          icon={QrCode}
          accent="warning"
        />
      </div>

      {/* Rincian keuangan: pisahkan uang penjualan, layanan, dan pajak. */}
      <div>
        <h3 className="mb-3 font-display text-sm font-semibold text-text">Rincian Keuangan</h3>
        <div className="grid gap-4 sm:grid-cols-3">
          <StatCard
            label="Penjualan"
            value={formatIDR(summary?.salesTotal ?? 0)}
            icon={ShoppingBag}
            accent="primary"
          />
          <StatCard
            label="Layanan"
            value={formatIDR(summary?.serviceTotal ?? 0)}
            icon={Sparkles}
            accent="info"
          />
          <StatCard
            label="Pajak (PPN)"
            value={formatIDR(summary?.taxTotal ?? 0)}
            icon={Landmark}
            accent="warning"
          />
        </div>
      </div>

      <MonthComparisonCard
        loading={thisMonthQuery.loading || lastMonthQuery.loading}
        error={thisMonthQuery.error || lastMonthQuery.error}
        thisMonthLabel={monthRanges.thisLabel}
        lastMonthLabel={monthRanges.lastLabel}
        thisRevenue={thisMonthQuery.data?.summary.revenue ?? 0}
        lastRevenue={lastMonthQuery.data?.summary.revenue ?? 0}
      />

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader className="pb-2">
            <CardTitle className="font-display">Tren Penjualan</CardTitle>
            <CardDescription>Pendapatan & transaksi harian, 30 hari terakhir</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-72">
              <ChartState
                loading={salesQuery.loading}
                error={salesQuery.error}
                empty={salesData.length === 0}
              >
                <ResponsiveContainer>
                  <AreaChart data={salesData} margin={{ top: 8, right: 8, left: 4, bottom: 0 }}>
                    <defs>
                      <linearGradient id="gSales" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor={colors.primary} stopOpacity={0.28} />
                        <stop offset="100%" stopColor={colors.primary} stopOpacity={0.02} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid vertical={false} stroke={colors.border} strokeOpacity={0.7} />
                    <XAxis
                      dataKey="day"
                      tickFormatter={formatDayShort}
                      tickLine={false}
                      axisLine={false}
                      tickMargin={10}
                      minTickGap={28}
                      tick={AXIS_TICK}
                    />
                    <YAxis
                      yAxisId="left"
                      tickFormatter={formatCompactIDR}
                      tickLine={false}
                      axisLine={false}
                      width={64}
                      tick={AXIS_TICK}
                    />
                    <YAxis
                      yAxisId="right"
                      orientation="right"
                      tickLine={false}
                      axisLine={false}
                      allowDecimals={false}
                      width={32}
                      tick={AXIS_TICK}
                    />
                    <Tooltip
                      cursor={{ stroke: colors.primary, strokeOpacity: 0.25, strokeWidth: 1.5 }}
                      content={
                        <ChartTooltip
                          labelFormatter={(l) => formatDayShort(String(l))}
                          formatter={(v, name) =>
                            name === "Transaksi" ? formatNumber(v) : formatIDR(v)
                          }
                        />
                      }
                    />
                    <Area
                      yAxisId="left"
                      type="monotone"
                      dataKey="revenue"
                      name="Pendapatan"
                      stroke={colors.primary}
                      strokeWidth={2.25}
                      fill="url(#gSales)"
                      dot={false}
                      activeDot={{
                        r: 4,
                        strokeWidth: 2,
                        stroke: colors.surface,
                        fill: colors.primary,
                      }}
                    />
                    <Area
                      yAxisId="right"
                      type="monotone"
                      dataKey="txCount"
                      name="Transaksi"
                      stroke={colors.secondary}
                      strokeWidth={1.75}
                      strokeDasharray="5 3"
                      fill="none"
                      dot={false}
                      activeDot={{ r: 3, strokeWidth: 0, fill: colors.secondary }}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </ChartState>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="font-display">Metode Pembayaran</CardTitle>
            <CardDescription>Tunai vs QRIS</CardDescription>
          </CardHeader>
          <CardContent>
            <ChartState
              loading={paymentQuery.loading}
              error={paymentQuery.error}
              empty={paymentTotal === 0}
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

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="font-display">Produk Terlaris</CardTitle>
            <CardDescription>Berdasarkan unit terjual</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-64">
              <ChartState
                loading={topProductsQuery.loading}
                error={topProductsQuery.error}
                empty={topProducts.length === 0}
              >
                <ResponsiveContainer>
                  <BarChart data={topProducts} layout="vertical" margin={{ left: 8, right: 28 }}>
                    <CartesianGrid horizontal={false} stroke={colors.border} strokeOpacity={0.7} />
                    <XAxis type="number" allowDecimals={false} hide />
                    <YAxis
                      dataKey="name"
                      type="category"
                      tickLine={false}
                      axisLine={false}
                      width={120}
                      tick={{ fill: colors.muted, fontSize: 11 }}
                    />
                    <Tooltip
                      cursor={{ fill: colors.surfaceMuted, fillOpacity: 0.6 }}
                      content={<ChartTooltip formatter={(v) => formatNumber(v)} />}
                    />
                    <Bar dataKey="qty" name="Unit terjual" radius={[0, 4, 4, 0]} maxBarSize={22}>
                      {topProducts.map((p, i) => (
                        <Cell
                          key={p.name}
                          fill={colors.primary}
                          fillOpacity={rankShade(i, topProducts.length)}
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

        <Card>
          <CardHeader>
            <CardTitle className="font-display">Penjualan per Kategori</CardTitle>
            <CardDescription>Pembagian pendapatan</CardDescription>
          </CardHeader>
          <CardContent>
            <ChartState
              loading={categoryQuery.loading}
              error={categoryQuery.error}
              empty={categories.length === 0}
            >
              <div className="space-y-3.5">
                {categories.map((c, i) => (
                  <div key={c.category}>
                    <div className="mb-1.5 flex items-center justify-between gap-2 text-xs">
                      <span className="font-medium text-text">{c.category}</span>
                      <span className="font-mono tabular-nums text-muted">
                        {formatIDR(c.revenue)}
                      </span>
                    </div>
                    <div className="h-2 overflow-hidden rounded-full bg-surface-muted">
                      <div
                        className="h-full rounded-full bg-primary"
                        style={{
                          width: `${(c.revenue / categoryMax) * 100}%`,
                          opacity: rankShade(i, categories.length),
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

      <Card>
        <CardHeader>
          <CardTitle className="font-display">Transaksi Terbaru</CardTitle>
          <CardDescription>Transaksi terakhir di toko Anda</CardDescription>
        </CardHeader>
        <CardContent>
          <ChartState
            loading={dashboardQuery.loading}
            error={dashboardQuery.error}
            empty={recent.length === 0}
          >
            <div className="space-y-1">
              {recent.map((t) => {
                const Icon: LucideIcon = t.paymentMethod === "qris" ? QrCode : Wallet;
                return (
                  <div
                    key={t.id}
                    className="flex items-start gap-3 rounded-lg p-2.5 transition-colors hover:bg-surface-muted"
                  >
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-soft text-primary">
                      <Icon className="h-4 w-4" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-baseline justify-between gap-2">
                        <p className="font-mono text-sm font-medium text-text">{t.code}</p>
                        <span className="shrink-0 font-mono text-sm font-semibold tabular-nums text-text">
                          {formatIDR(t.total)}
                        </span>
                      </div>
                      <div className="mt-0.5 flex flex-wrap items-baseline justify-between gap-2">
                        <p className="text-xs text-muted">
                          {sourceLabel[t.source] ?? t.source} ·{" "}
                          {methodLabel[t.paymentMethod] ?? t.paymentMethod}
                        </p>
                        <span className="shrink-0 text-xs text-muted">
                          {formatDateTime(t.createdAt)}
                        </span>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </ChartState>
        </CardContent>
      </Card>
    </div>
  );
}
