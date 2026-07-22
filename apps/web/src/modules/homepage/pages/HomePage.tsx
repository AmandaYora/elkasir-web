import { Link } from "react-router-dom";
import {
  BarChart3,
  Check,
  CheckCircle2,
  ChevronRight,
  Mail,
  QrCode,
  ShieldCheck,
  UtensilsCrossed,
  Wallet,
} from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { HomepageNav } from "@/modules/homepage/components/HomepageNav";
import { HomepageFooter } from "@/modules/homepage/components/HomepageFooter";
import "@/modules/homepage/styles.css";

const STATUS_STYLE = {
  baru: {
    label: "Baru",
    classes: "bg-(--hp-warning)/10 text-(--hp-warning) border border-(--hp-warning)/20",
  },
  diproses: {
    label: "Diproses",
    classes: "bg-(--hp-line)/30 text-(--hp-ink-soft) border border-(--hp-line)",
  },
  lunas: {
    label: "Lunas",
    classes: "bg-(--hp-success)/10 text-(--hp-success) border border-(--hp-success)/20",
  },
} as const;

const TICKETS: {
  table: string;
  status: keyof typeof STATUS_STYLE;
  items: string;
  price: string;
}[] = [
  {
    table: "Meja 04",
    status: "baru",
    items: "2 item · Ayam Geprek, Es Teh",
    price: "Rp47.000",
  },
  {
    table: "Meja 02",
    status: "diproses",
    items: "4 item · Nasi Goreng x2, Jus Alpukat, Kerupuk",
    price: "Rp132.000",
  },
  {
    table: "Meja 07",
    status: "lunas",
    items: "1 item · Kopi Susu",
    price: "Rp18.000",
  },
];

const FEATURES = [
  {
    icon: QrCode,
    name: "Self-Order Pelanggan",
    detail:
      "Pelanggan bisa memesan dan membayar langsung dari meja dengan scan QR. Tanpa antre, tanpa perlu instal aplikasi apa pun.",
  },
  {
    icon: Wallet,
    name: "Manajemen Kasir & Shift",
    detail:
      "Kelola pergantian shift karyawan dengan pencatatan pemasukan kas yang akurat, rapi, dan mudah dipantau.",
  },
  {
    icon: BarChart3,
    name: "Laporan Penjualan",
    detail:
      "Pantau omzet harian, menu paling laris, dan ringkasan pendapatan langsung dari HP atau laptop Anda.",
  },
  {
    icon: UtensilsCrossed,
    name: "Kelola Menu & Stok",
    detail:
      "Atur daftar menu, ketersediaan stok, dan perubahan harga dengan sangat mudah dari satu layar utama.",
  },
];

const STEPS = [
  {
    n: "01",
    title: "Pelanggan Scan QR",
    detail: "Kode QR tersedia di setiap meja. Pelanggan cukup scan menggunakan kamera HP mereka.",
  },
  {
    n: "02",
    title: "Pilih Pesanan",
    detail:
      "Daftar menu digital terbuka secara instan. Pelanggan bisa melihat gambar dan harga dengan jelas.",
  },
  {
    n: "03",
    title: "Bayar Praktis",
    detail: "Pembayaran langsung menggunakan QRIS dari aplikasi m-banking atau e-wallet apa saja.",
  },
  {
    n: "04",
    title: "Pesanan Disiapkan",
    detail:
      "Pesanan yang sudah dibayar otomatis masuk ke layar kasir atau dapur untuk segera dibuatkan.",
  },
];

const CONSULT_POINTS = [
  "Bantuan penuh saat mendaftarkan menu dan data awal toko Anda",
  "Pilihan paket harga yang fleksibel sesuai kebutuhan jumlah cabang",
  "Layanan prioritas jika Anda mengalami kendala saat jam operasional",
];

export default function HomePage() {
  return (
    <div className="hp-root relative">
      <HomepageNav />

      {/* ── Hero Section ── */}
      <section className="relative z-10 overflow-hidden border-b border-(--hp-line) bg-white pt-16 pb-20 sm:pt-24 sm:pb-32">
        <div className="mx-auto grid max-w-7xl gap-16 px-6 lg:grid-cols-2 lg:items-center">
          <div className="hp-animate-fade flex flex-col items-start text-left">
            <div className="inline-flex items-center gap-2 rounded-full border border-(--hp-line) bg-(--hp-surface-raised) px-4 py-1.5">
              <span className="flex h-2 w-2 rounded-full bg-(--hp-primary)" aria-hidden="true" />
              <p className="hp-font-mono text-xs font-medium tracking-widest text-(--hp-ink-soft) uppercase">
                Sistem Kasir &amp; Pemesanan Cerdas
              </p>
            </div>

            <h1 className="hp-font-display mt-8 text-[3rem] leading-[1.1] font-bold tracking-tight text-(--hp-ink) sm:text-6xl md:text-7xl">
              Setiap meja, <br />
              kasir sendiri.
            </h1>

            <p className="mt-6 max-w-lg text-lg leading-relaxed text-(--hp-ink-soft)">
              Ubah cara pelanggan Anda memesan. Dari memilih menu hingga membayar, semuanya bisa
              dilakukan langsung dari meja tanpa harus mengantre.
            </p>

            <div className="mt-10 flex flex-col gap-4 sm:flex-row sm:items-center">
              <a
                href="#konsultasi"
                className="inline-flex items-center justify-center gap-2 rounded-md bg-(--hp-primary) px-6 py-3 text-sm font-medium text-white transition-colors hover:bg-(--hp-primary-hover) focus:outline-none"
              >
                Konsultasi Harga
                <ChevronRight className="h-4 w-4" aria-hidden="true" />
              </a>
              <a
                href="#cara-kerja"
                className="inline-flex items-center justify-center rounded-md border border-(--hp-line) bg-white px-6 py-3 text-sm font-medium text-(--hp-ink) transition-colors hover:bg-(--hp-surface-raised) focus:outline-none"
              >
                Cara Kerjanya
              </a>
            </div>

            <div className="mt-10 flex items-center gap-3 rounded-lg border border-(--hp-line) bg-(--hp-surface-raised) px-4 py-3">
              <div className="flex items-center justify-center">
                <ShieldCheck className="h-5 w-5 text-(--hp-primary)" aria-hidden="true" />
              </div>
              <p className="hp-font-mono text-[11px] font-semibold tracking-wide text-(--hp-ink-soft)">
                TRANSAKSI AMAN &amp; TERPERCAYA UNTUK SETIAP PESANAN
              </p>
            </div>
          </div>

          {/* Clean Dashboard Mockup instead of floating tickets */}
          <div className="hp-animate-fade relative flex flex-col gap-4 rounded-xl border border-(--hp-line) bg-(--hp-surface-raised) p-6 sm:p-8">
            <div className="mb-2 flex items-center justify-between">
              <p className="hp-font-display font-semibold text-(--hp-ink)">Live Orders</p>
              <div className="flex items-center gap-2">
                <span className="relative flex h-2 w-2">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-(--hp-success) opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-2 w-2 bg-(--hp-success)"></span>
                </span>
                <span className="text-xs font-medium text-(--hp-ink-soft)">Real-time</span>
              </div>
            </div>
            <div className="flex flex-col gap-4">
              {TICKETS.map((t) => {
                const status = STATUS_STYLE[t.status];
                return (
                  <div key={t.table} className="hp-card flex flex-col gap-3 p-5">
                    <div className="flex items-center justify-between">
                      <span className="hp-font-mono text-sm font-semibold tracking-wide text-(--hp-ink)">
                        {t.table}
                      </span>
                      <span
                        className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 hp-font-mono text-[10px] font-bold tracking-widest uppercase ${status.classes}`}
                      >
                        {t.status === "lunas" && <Check className="h-3 w-3" aria-hidden="true" />}
                        {status.label}
                      </span>
                    </div>
                    <p className="text-sm font-medium leading-relaxed text-(--hp-ink-soft)">
                      {t.items}
                    </p>
                    <p className="hp-font-mono mt-1 text-lg font-bold tracking-tight text-(--hp-ink)">
                      {t.price}
                    </p>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </section>

      {/* ── Fitur (Clean Grid) ── */}
      <section
        id="produk"
        className="relative z-10 bg-(--hp-surface-raised) px-6 py-24 border-b border-(--hp-line)"
      >
        <div className="mx-auto max-w-7xl">
          <div className="text-center">
            <h2 className="hp-font-display text-3xl font-bold tracking-tight text-(--hp-ink) sm:text-4xl">
              Satu sistem untuk mengelola segalanya
            </h2>
            <p className="mt-4 text-lg text-(--hp-ink-soft)">
              Pantau meja, kasir, dan laporan dengan mudah dan praktis.
            </p>
          </div>

          <div className="mt-16 grid gap-6 sm:grid-cols-2 lg:grid-cols-2">
            {FEATURES.map((f) => (
              <div key={f.name} className="hp-card p-8">
                <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-(--hp-primary-soft) text-(--hp-primary)">
                  <f.icon className="h-6 w-6" aria-hidden="true" />
                </div>
                <div className="mt-6">
                  <h3 className="hp-font-display text-xl font-semibold text-(--hp-ink)">
                    {f.name}
                  </h3>
                  <p className="mt-3 text-[15px] leading-relaxed text-(--hp-ink-soft)">
                    {f.detail}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ── Cara kerja ── */}
      <section
        id="cara-kerja"
        className="relative z-10 px-6 py-24 bg-white border-b border-(--hp-line)"
      >
        <div className="mx-auto max-w-7xl">
          <div className="text-center">
            <h2 className="hp-font-display text-3xl font-bold tracking-tight text-(--hp-ink) sm:text-4xl">
              Pengalaman pemesanan yang cepat
            </h2>
            <p className="mt-4 text-lg text-(--hp-ink-soft)">
              Hanya 4 langkah mudah dari duduk hingga pesanan disiapkan.
            </p>
          </div>

          <div className="mt-16 relative">
            {/* Connecting Line (Desktop) */}
            <div className="hidden lg:block absolute top-8 left-[10%] right-[10%] h-[1px] bg-(--hp-line)" />

            <ol className="grid gap-12 sm:grid-cols-2 lg:grid-cols-4 lg:gap-8">
              {STEPS.map((s) => (
                <li key={s.n} className="relative flex flex-col items-center text-center bg-white">
                  <div className="relative z-10 flex h-16 w-16 items-center justify-center rounded-full bg-(--hp-surface-raised) border border-(--hp-line)">
                    <span className="hp-font-mono text-xl font-bold text-(--hp-ink)">{s.n}</span>
                  </div>
                  <h3 className="hp-font-display mt-6 text-lg font-bold text-(--hp-ink)">
                    {s.title}
                  </h3>
                  <p className="mt-3 max-w-[28ch] text-sm leading-relaxed text-(--hp-ink-soft)">
                    {s.detail}
                  </p>
                </li>
              ))}
            </ol>
          </div>
        </div>
      </section>

      {/* ── Harga & Kemitraan (Professional CTA) ── */}
      <section
        id="konsultasi"
        className="relative z-10 px-6 py-24 sm:py-32 bg-(--hp-surface-raised)"
      >
        <div className="mx-auto max-w-5xl">
          <div className="hp-card overflow-hidden rounded-2xl bg-white">
            <div className="grid gap-12 px-8 py-12 sm:px-12 lg:grid-cols-[1.3fr_0.7fr] lg:gap-8 lg:p-16">
              <div>
                <h2 className="hp-font-display text-3xl font-bold tracking-tight text-(--hp-ink)">
                  Solusi kasir lengkap tanpa biaya tersembunyi.
                </h2>
                <p className="mt-4 text-lg leading-relaxed text-(--hp-ink-soft)">
                  Kebutuhan bisnis tiap tempat berbeda. Mari bicarakan kendala operasional Anda dan
                  temukan solusinya bersama kami.
                </p>

                <ul className="mt-8 space-y-4">
                  {CONSULT_POINTS.map((point) => (
                    <li
                      key={point}
                      className="flex items-start gap-4 text-sm leading-relaxed text-(--hp-ink)"
                    >
                      <div className="mt-1 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-(--hp-primary-soft)">
                        <CheckCircle2
                          className="h-3.5 w-3.5 text-(--hp-primary)"
                          aria-hidden="true"
                        />
                      </div>
                      {point}
                    </li>
                  ))}
                </ul>
              </div>

              <div className="flex flex-col justify-center items-center lg:items-end text-center lg:text-right border-t border-(--hp-line) lg:border-t-0 lg:border-l pt-12 lg:pt-0 lg:pl-12">
                <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-(--hp-surface-raised) border border-(--hp-line)">
                  <Mail className="h-6 w-6 text-(--hp-ink)" aria-hidden="true" />
                </div>
                <h3 className="mt-6 text-lg font-semibold text-(--hp-ink)">Hubungi Tim Kami</h3>
                <a
                  href="mailto:cs@elcodelabs.com"
                  className="hp-font-mono mt-2 block text-lg font-bold text-(--hp-ink) transition-colors hover:text-(--hp-primary)"
                >
                  cs@elcodelabs.com
                </a>
                <a
                  href="mailto:cs@elcodelabs.com"
                  className="mt-6 inline-flex items-center justify-center gap-2 rounded-md bg-(--hp-ink) px-6 py-3 text-sm font-medium text-white transition-colors hover:bg-slate-800"
                >
                  Kirim Email
                  <ChevronRight className="h-4 w-4" aria-hidden="true" />
                </a>
              </div>
            </div>
          </div>

          <p className="mt-12 text-center text-sm font-medium text-(--hp-ink-soft)">
            Informasi legal mengenai layanan kami ada di{" "}
            <Link
              to={ROUTE_PATHS.homepageTerms}
              className="text-(--hp-ink) underline underline-offset-4 hover:text-(--hp-primary) transition-colors"
            >
              Syarat &amp; Ketentuan
            </Link>
          </p>
        </div>
      </section>

      <HomepageFooter />
    </div>
  );
}
