import {
  DollarSign,
  Receipt,
  Wallet,
  QrCode,
  Loader2,
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
} from "recharts";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/shared/components/ui/card";
import { formatIDR, formatDateTime } from "@/shared/lib/formatter";
import { useAsync } from "@/shared/hooks/useAsync";
import { useAuthStore } from "@/shared/stores/auth.store";
import { colors, chartPalette } from "@/theme";
import { StatCard } from "@/modules/dashboard/components/StatCard";
import { dashboardService } from "@/modules/dashboard/services/dashboard.service";

type ChartTooltipPayload = {
  color?: string;
  name?: string;
  value: number;
};

function ChartTooltip({
  active,
  payload,
  label,
  formatter,
}: {
  active?: boolean;
  payload?: ChartTooltipPayload[];
  label?: string;
  formatter?: (value: number, name?: string) => string;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-lg border border-border bg-surface px-3 py-2 text-xs shadow-md">
      {label && <div className="mb-1 font-medium">{label}</div>}
      {payload.map((p, i) => (
        <div key={i} className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full" style={{ background: p.color }} />
          <span className="text-muted">{p.name}:</span>
          <span className="font-medium">{formatter ? formatter(p.value, p.name) : p.value}</span>
        </div>
      ))}
    </div>
  );
}

function ChartState({
  loading,
  error,
  empty,
  className = "h-72",
}: {
  loading: boolean;
  error: string | null;
  empty: boolean;
  className?: string;
}) {
  return (
    <div className={`flex ${className} items-center justify-center text-sm`}>
      {loading ? (
        <Loader2 className="h-5 w-5 animate-spin text-muted" />
      ) : error ? (
        <span className="text-danger">{error}</span>
      ) : empty ? (
        <span className="text-muted">Belum ada data.</span>
      ) : null}
    </div>
  );
}

const sourceLabel: Record<string, string> = {
  cashier: "Kasir",
  self_order: "Pesan mandiri",
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

  const summary = dashboardQuery.data?.summary;
  const recent = dashboardQuery.data?.recent ?? [];

  const salesData = (salesQuery.data ?? []).map((d) => ({
    day: d.day,
    revenue: d.revenue,
    txCount: d.txCount,
  }));

  const payment = paymentQuery.data;
  const paymentSlices = payment
    ? [
        { name: "Tunai", value: payment.cashTotal, color: chartPalette[0] },
        { name: "QRIS", value: payment.qrisTotal, color: chartPalette[1] },
      ]
    : [];
  const paymentTotal = (payment?.cashTotal ?? 0) + (payment?.qrisTotal ?? 0);

  const topProducts = (topProductsQuery.data ?? []).map((p) => ({
    name: p.productName,
    qty: p.qty,
    revenue: p.revenue,
  }));

  const categories = categoryQuery.data ?? [];
  const categoryMax = Math.max(0, ...categories.map((c) => c.revenue));

  return (
    <div className="space-y-6 p-4 md:p-6">
      <div>
        <h2 className="text-lg font-semibold text-text">Selamat datang, {firstName}</h2>
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
          value={String(summary?.txCount ?? 0)}
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
        <h3 className="mb-3 text-sm font-semibold text-text">Rincian Keuangan</h3>
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

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader className="pb-2">
            <CardTitle>Tren Penjualan</CardTitle>
            <CardDescription>Pendapatan harian, 30 hari terakhir</CardDescription>
          </CardHeader>
          <CardContent>
            {salesQuery.loading || salesQuery.error || salesData.length === 0 ? (
              <ChartState
                loading={salesQuery.loading}
                error={salesQuery.error}
                empty={salesData.length === 0}
              />
            ) : (
              <div className="h-72">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={salesData} margin={{ top: 10, right: 20, left: 0, bottom: 0 }}>
                    <defs>
                      <linearGradient id="gSales" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor={chartPalette[0]} stopOpacity={0.3} />
                        <stop offset="100%" stopColor={chartPalette[0]} stopOpacity={0} />
                      </linearGradient>
                      <linearGradient id="gTxn" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor={chartPalette[2]} stopOpacity={0.2} />
                        <stop offset="100%" stopColor={chartPalette[2]} stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid
                      vertical={false}
                      strokeDasharray="3 3"
                      stroke={colors.border}
                      opacity={0.6}
                    />
                    <XAxis
                      dataKey="day"
                      tickLine={false}
                      axisLine={false}
                      stroke={colors.muted}
                      fontSize={11}
                      dy={6}
                    />
                    <YAxis
                      yAxisId="left"
                      tickLine={false}
                      axisLine={false}
                      stroke={colors.muted}
                      fontSize={11}
                      tickFormatter={(v) => `${(v / 1_000_000).toFixed(1)}M`}
                      width={50}
                    />
                    <YAxis
                      yAxisId="right"
                      orientation="right"
                      tickLine={false}
                      axisLine={false}
                      stroke={colors.muted}
                      fontSize={11}
                      width={40}
                    />
                    <Tooltip
                      content={
                        <ChartTooltip
                          formatter={(v: number, name?: string) =>
                            name === "Transaksi" ? `${v}` : formatIDR(v)
                          }
                        />
                      }
                    />
                    <Area
                      yAxisId="left"
                      type="monotone"
                      dataKey="revenue"
                      name="Pendapatan"
                      stroke={chartPalette[0]}
                      strokeWidth={2.5}
                      fill="url(#gSales)"
                      activeDot={{ r: 4, strokeWidth: 0 }}
                    />
                    <Area
                      yAxisId="right"
                      type="monotone"
                      dataKey="txCount"
                      name="Transaksi"
                      stroke={chartPalette[2]}
                      strokeWidth={2}
                      strokeDasharray="5 3"
                      fill="url(#gTxn)"
                      activeDot={{ r: 3, strokeWidth: 0 }}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Metode Pembayaran</CardTitle>
            <CardDescription>Tunai vs QRIS</CardDescription>
          </CardHeader>
          <CardContent>
            {paymentQuery.loading || paymentQuery.error || paymentTotal === 0 ? (
              <ChartState
                loading={paymentQuery.loading}
                error={paymentQuery.error}
                empty={paymentTotal === 0}
                className="h-44"
              />
            ) : (
              <>
                <div className="h-44">
                  <ResponsiveContainer>
                    <PieChart>
                      <Pie
                        data={paymentSlices}
                        dataKey="value"
                        innerRadius={50}
                        outerRadius={75}
                        paddingAngle={3}
                        stroke="none"
                      >
                        {paymentSlices.map((e, i) => (
                          <Cell key={i} fill={e.color} />
                        ))}
                      </Pie>
                      <Tooltip content={<ChartTooltip formatter={(v: number) => formatIDR(v)} />} />
                    </PieChart>
                  </ResponsiveContainer>
                </div>
                <div className="mt-3 space-y-2">
                  {paymentSlices.map((p) => (
                    <div key={p.name} className="flex items-center justify-between text-sm">
                      <div className="flex items-center gap-2">
                        <span
                          className="h-2.5 w-2.5 rounded-full"
                          style={{ background: p.color }}
                        />
                        <span>{p.name}</span>
                      </div>
                      <span className="font-medium">
                        {formatIDR(p.value)}{" "}
                        <span className="text-xs text-muted">
                          ({Math.round((p.value / paymentTotal) * 100)}%)
                        </span>
                      </span>
                    </div>
                  ))}
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Produk Terlaris</CardTitle>
            <CardDescription>Berdasarkan unit terjual</CardDescription>
          </CardHeader>
          <CardContent>
            {topProductsQuery.loading || topProductsQuery.error || topProducts.length === 0 ? (
              <ChartState
                loading={topProductsQuery.loading}
                error={topProductsQuery.error}
                empty={topProducts.length === 0}
                className="h-64"
              />
            ) : (
              <div className="h-64">
                <ResponsiveContainer>
                  <BarChart data={topProducts} layout="vertical" margin={{ left: 20 }}>
                    <CartesianGrid
                      horizontal={false}
                      strokeDasharray="3 3"
                      stroke={colors.border}
                    />
                    <XAxis
                      type="number"
                      tickLine={false}
                      axisLine={false}
                      fontSize={11}
                      stroke={colors.muted}
                      allowDecimals={false}
                    />
                    <YAxis
                      dataKey="name"
                      type="category"
                      tickLine={false}
                      axisLine={false}
                      fontSize={11}
                      stroke={colors.muted}
                      width={110}
                    />
                    <Tooltip content={<ChartTooltip />} />
                    <Bar
                      dataKey="qty"
                      name="Unit terjual"
                      fill={chartPalette[1]}
                      radius={[0, 6, 6, 0]}
                    />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Penjualan per Kategori</CardTitle>
            <CardDescription>Pembagian pendapatan</CardDescription>
          </CardHeader>
          <CardContent>
            {categoryQuery.loading || categoryQuery.error || categories.length === 0 ? (
              <ChartState
                loading={categoryQuery.loading}
                error={categoryQuery.error}
                empty={categories.length === 0}
                className="h-40"
              />
            ) : (
              <div className="space-y-3.5">
                {categories.map((c, i) => {
                  const pct = categoryMax > 0 ? (c.revenue / categoryMax) * 100 : 0;
                  return (
                    <div key={c.category}>
                      <div className="mb-1.5 flex items-center justify-between text-xs">
                        <span className="font-medium">{c.category}</span>
                        <span className="text-muted">{formatIDR(c.revenue)}</span>
                      </div>
                      <div className="h-2 overflow-hidden rounded-full bg-surface-muted">
                        <div
                          className="h-full rounded-full"
                          style={{
                            width: `${pct}%`,
                            background: chartPalette[i % chartPalette.length],
                          }}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Transaksi Terbaru</CardTitle>
          <CardDescription>Transaksi terakhir di toko Anda</CardDescription>
        </CardHeader>
        <CardContent>
          {dashboardQuery.loading || dashboardQuery.error || recent.length === 0 ? (
            <ChartState
              loading={dashboardQuery.loading}
              error={dashboardQuery.error}
              empty={recent.length === 0}
              className="h-40"
            />
          ) : (
            <div className="space-y-1">
              {recent.map((t) => {
                const Icon: LucideIcon = t.paymentMethod === "qris" ? QrCode : Wallet;
                return (
                  <div
                    key={t.id}
                    className="flex items-start gap-3 rounded-lg p-2.5 transition-colors hover:bg-surface-muted"
                  >
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                      <Icon className="h-4 w-4" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-baseline justify-between gap-2">
                        <p className="font-mono text-sm font-medium">{t.code}</p>
                        <span className="shrink-0 text-sm font-semibold tabular-nums">
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
          )}
        </CardContent>
      </Card>
    </div>
  );
}
