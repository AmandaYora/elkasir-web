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
  Legend,
} from "recharts";
import { Loader2 } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/shared/components/ui/card";
import { formatIDR } from "@/shared/lib/formatter";
import { useAsync } from "@/shared/hooks/useAsync";
import { cn } from "@/shared/lib/cn";
import { colors, chartPalette } from "@/theme";
import { statisticsService } from "@/modules/statistics/services/statistics.service";

type ChartTooltipPayload = {
  color?: string;
  fill?: string;
  name?: string;
  value: number;
};

function CT({
  active,
  payload,
  label,
  fmt,
}: {
  active?: boolean;
  payload?: ChartTooltipPayload[];
  label?: string;
  fmt?: (value: number) => string;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-lg border border-border bg-surface px-3 py-2 text-xs shadow-md">
      {label && <div className="mb-1 font-medium">{label}</div>}
      {payload.map((p, i) => (
        <div key={i} className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full" style={{ background: p.color || p.fill }} />
          <span className="text-muted">{p.name}:</span>
          <span className="font-medium">{fmt ? fmt(p.value) : p.value}</span>
        </div>
      ))}
    </div>
  );
}

// Pembungkus status loading/error/kosong untuk isi chart.
function ChartState({
  isLoading,
  error,
  isEmpty,
  children,
}: {
  isLoading: boolean;
  error: string | null;
  isEmpty: boolean;
  children: React.ReactNode;
}) {
  if (isLoading)
    return (
      <div className="flex h-full items-center justify-center text-muted">
        <Loader2 className="h-5 w-5 animate-spin" />
      </div>
    );
  if (error)
    return (
      <div className="flex h-full items-center justify-center text-center text-sm text-danger">
        Gagal memuat data. {error}
      </div>
    );
  if (isEmpty)
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted">Belum ada data</div>
    );
  return <>{children}</>;
}

const RANGE_OPTIONS = [
  { value: "7", label: "7 hari" },
  { value: "30", label: "30 hari" },
  { value: "90", label: "90 hari" },
];

// "YYYY-MM-DD" untuk hari ini dan hari ini − n.
function ymd(d: Date) {
  return d.toISOString().slice(0, 10);
}
function daysAgo(n: number) {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d;
}

export default function StatisticsPage() {
  const [range, setRange] = useState("30");
  const days = Number(range);

  const period = useMemo(
    () => ({ from: ymd(daysAgo(days)), to: ymd(new Date()) }),
    [days],
  );

  const salesQuery = useAsync(() => statisticsService.sales(period), [period.from, period.to]);
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

  const paymentData = useMemo(() => {
    if (!payment) return [];
    const rows = [
      { name: "Tunai", value: payment.cashTotal, color: chartPalette[0] },
      { name: "QRIS", value: payment.qrisTotal, color: chartPalette[1] },
    ];
    return rows.filter((r) => r.value > 0);
  }, [payment]);

  const staffRanked = useMemo(() => [...staff].sort((a, b) => b.revenue - a.revenue), [staff]);
  const staffMax = Math.max(1, ...staffRanked.map((s) => s.revenue));

  return (
    <div className="space-y-6 p-4 md:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold text-text">Statistik</h2>
          <p className="text-sm text-muted">Analitik mendalam untuk seluruh bisnis Anda.</p>
        </div>
        <div className="inline-flex rounded-md border border-border bg-surface p-0.5">
          {RANGE_OPTIONS.map((o) => (
            <button
              key={o.value}
              type="button"
              onClick={() => setRange(o.value)}
              className={cn(
                "rounded px-3 py-1 text-xs font-medium transition-colors",
                range === o.value
                  ? "bg-primary text-primary-foreground"
                  : "text-muted hover:bg-surface-muted",
              )}
            >
              {o.label}
            </button>
          ))}
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Pendapatan</CardTitle>
            <CardDescription>Pendapatan harian</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-72">
              <ChartState isLoading={salesQuery.loading} error={salesQuery.error} isEmpty={sales.length === 0}>
                <ResponsiveContainer>
                  <AreaChart data={sales}>
                    <defs>
                      <linearGradient id="rv" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor={chartPalette[0]} stopOpacity={0.35} />
                        <stop offset="100%" stopColor={chartPalette[0]} stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid vertical={false} strokeDasharray="3 3" stroke={colors.border} />
                    <XAxis
                      dataKey="day"
                      tickLine={false}
                      axisLine={false}
                      fontSize={11}
                      stroke={colors.muted}
                    />
                    <YAxis
                      tickLine={false}
                      axisLine={false}
                      fontSize={11}
                      stroke={colors.muted}
                      tickFormatter={(v) => `${(v / 1_000_000).toFixed(0)}M`}
                    />
                    <Tooltip content={<CT fmt={(v: number) => formatIDR(v)} />} />
                    <Legend iconType="circle" wrapperStyle={{ fontSize: 12 }} />
                    <Area
                      type="monotone"
                      dataKey="revenue"
                      name="Pendapatan"
                      stroke={chartPalette[0]}
                      strokeWidth={2.5}
                      fill="url(#rv)"
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </ChartState>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Volume Transaksi</CardTitle>
            <CardDescription>Jumlah transaksi harian</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-72">
              <ChartState isLoading={salesQuery.loading} error={salesQuery.error} isEmpty={sales.length === 0}>
                <ResponsiveContainer>
                  <BarChart data={sales}>
                    <CartesianGrid vertical={false} strokeDasharray="3 3" stroke={colors.border} />
                    <XAxis
                      dataKey="day"
                      tickLine={false}
                      axisLine={false}
                      fontSize={11}
                      stroke={colors.muted}
                    />
                    <YAxis
                      tickLine={false}
                      axisLine={false}
                      fontSize={11}
                      stroke={colors.muted}
                      allowDecimals={false}
                    />
                    <Tooltip content={<CT />} />
                    <Bar dataKey="txCount" name="Transaksi" fill={chartPalette[1]} radius={[6, 6, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </ChartState>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle>Distribusi Pembayaran</CardTitle>
            <CardDescription>Proporsi pendapatan per metode</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-56">
              <ChartState
                isLoading={paymentQuery.loading}
                error={paymentQuery.error}
                isEmpty={paymentData.length === 0}
              >
                <ResponsiveContainer>
                  <PieChart>
                    <Pie
                      data={paymentData}
                      dataKey="value"
                      nameKey="name"
                      innerRadius="45%"
                      outerRadius="80%"
                      paddingAngle={2}
                    >
                      {paymentData.map((e, i) => (
                        <Cell key={i} fill={e.color} />
                      ))}
                    </Pie>
                    <Tooltip content={<CT fmt={(v: number) => formatIDR(v)} />} />
                    <Legend iconType="circle" wrapperStyle={{ fontSize: 12 }} />
                  </PieChart>
                </ResponsiveContainer>
              </ChartState>
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Penjualan per Kategori</CardTitle>
            <CardDescription>Pendapatan menurut kategori menu</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-56">
              <ChartState
                isLoading={categoryQuery.loading}
                error={categoryQuery.error}
                isEmpty={categories.length === 0}
              >
                <ResponsiveContainer>
                  <BarChart data={categories}>
                    <CartesianGrid vertical={false} strokeDasharray="3 3" stroke={colors.border} />
                    <XAxis
                      dataKey="category"
                      tickLine={false}
                      axisLine={false}
                      fontSize={11}
                      stroke={colors.muted}
                    />
                    <YAxis
                      tickLine={false}
                      axisLine={false}
                      fontSize={11}
                      stroke={colors.muted}
                      tickFormatter={(v) => `${(v / 1_000_000).toFixed(0)}M`}
                    />
                    <Tooltip content={<CT fmt={(v: number) => formatIDR(v)} />} />
                    <Bar dataKey="revenue" name="Penjualan" fill={chartPalette[2]} radius={[6, 6, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </ChartState>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Produk Terlaris</CardTitle>
            <CardDescription>Unit terjual</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-64">
              <ChartState
                isLoading={topProductsQuery.loading}
                error={topProductsQuery.error}
                isEmpty={top.length === 0}
              >
                <ResponsiveContainer>
                  <BarChart data={top} layout="vertical" margin={{ left: 20 }}>
                    <CartesianGrid horizontal={false} strokeDasharray="3 3" stroke={colors.border} />
                    <XAxis
                      type="number"
                      tickLine={false}
                      axisLine={false}
                      fontSize={11}
                      stroke={colors.muted}
                      allowDecimals={false}
                    />
                    <YAxis
                      dataKey="productName"
                      type="category"
                      tickLine={false}
                      axisLine={false}
                      fontSize={11}
                      stroke={colors.muted}
                      width={110}
                    />
                    <Tooltip content={<CT />} />
                    <Bar dataKey="qty" name="Terjual" fill={chartPalette[3]} radius={[0, 6, 6, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </ChartState>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Kinerja Staf</CardTitle>
            <CardDescription>Peringkat pendapatan</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="min-h-[16rem]">
              <ChartState
                isLoading={staffQuery.loading}
                error={staffQuery.error}
                isEmpty={staffRanked.length === 0}
              >
                <div className="space-y-3">
                  {staffRanked.map((c, i) => (
                    <div key={c.staffId}>
                      <div className="mb-1.5 flex items-center justify-between text-sm">
                        <div className="flex items-center gap-2">
                          <span className="flex h-6 w-6 items-center justify-center rounded-full bg-surface-muted text-xs font-medium">
                            {i + 1}
                          </span>
                          <span className="font-medium">{c.name}</span>
                        </div>
                        <div className="text-right">
                          <div className="font-medium">{formatIDR(c.revenue)}</div>
                          <div className="text-xs text-muted">{c.txCount} trx</div>
                        </div>
                      </div>
                      <div className="h-2 overflow-hidden rounded-full bg-surface-muted">
                        <div
                          className="h-full rounded-full bg-primary"
                          style={{ width: `${(c.revenue / staffMax) * 100}%` }}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </ChartState>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
