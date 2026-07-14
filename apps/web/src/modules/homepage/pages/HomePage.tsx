import type { CSSProperties } from "react";
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

type TicketStyle = CSSProperties & { "--hp-delay"?: string; "--hp-tilt"?: string };

const STATUS_STYLE = {
  baru: { label: "Baru", classes: "bg-(--hp-amber)/15 text-(--hp-amber)" },
  diproses: { label: "Diproses", classes: "bg-white/10 text-(--hp-dark-ink-soft)" },
  lunas: { label: "Lunas", classes: "bg-(--hp-glow)/15 text-(--hp-glow)" },
} as const;

const TICKETS: {
  table: string;
  status: keyof typeof STATUS_STYLE;
  items: string;
  price: string;
  tilt: string;
}[] = [
  {
    table: "Meja 04",
    status: "baru",
    items: "2 item · Ayam Geprek, Es Teh",
    price: "Rp47.000",
    tilt: "-2deg",
  },
  {
    table: "Meja 02",
    status: "diproses",
    items: "4 item · Nasi Goreng x2, Jus Alpukat, Kerupuk",
    price: "Rp132.000",
    tilt: "1.5deg",
  },
  {
    table: "Meja 07",
    status: "lunas",
    items: "1 item · Kopi Susu",
    price: "Rp18.000",
    tilt: "-1deg",
  },
];

const FEATURES = [
  {
    icon: QrCode,
    name: "Self-Order QRIS Pelanggan",
    detail: "Pesan & bayar dari meja, tanpa antre kasir",
  },
  {
    icon: Wallet,
    name: "Kasir & Manajemen Shift",
    detail: "Buka-tutup shift, rekonsiliasi kas otomatis",
  },
  {
    icon: BarChart3,
    name: "Laporan & Analitik",
    detail: "Penjualan, produk terlaris, performa staf",
  },
  {
    icon: UtensilsCrossed,
    name: "Menu & Stok Terpusat",
    detail: "Ubah sekali, berlaku di semua kanal",
  },
];

const STEPS = [
  {
    n: "01",
    title: "Pindai kode QR di meja",
    detail:
      "Pelanggan memindai QR yang tertempel di meja restoran — tanpa perlu login atau instal aplikasi.",
  },
  {
    n: "02",
    title: "Pilih menu",
    detail: "Katalog produk toko terbuka langsung di browser HP pelanggan, lengkap dengan harga.",
  },
  {
    n: "03",
    title: "Bayar via QRIS",
    detail:
      "Kode QRIS muncul otomatis — pelanggan bayar dari aplikasi mobile banking/e-wallet apa pun.",
  },
  {
    n: "04",
    title: "Pesanan masuk ke kasir",
    detail: "Begitu lunas, pesanan otomatis muncul di sistem kasir toko untuk diproses.",
  },
];

const CONSULT_POINTS = [
  "Onboarding & migrasi data didampingi langsung oleh tim kami",
  "Skema penagihan disesuaikan dengan skala dan jumlah cabang",
  "Dukungan prioritas untuk kebutuhan operasional harian",
];

export default function HomePage() {
  return (
    <div className="hp-root">
      <HomepageNav />

      {/* ── Hero — the counter at the moment an order lands: the page's one signature
          scene, built from Elkasir's own self-order → kasir mechanic, not a stock visual. ── */}
      <section className="hp-dark-section relative overflow-hidden">
        <div
          aria-hidden="true"
          className="pointer-events-none absolute -top-32 -right-32 h-[480px] w-[480px] rounded-full bg-[radial-gradient(circle,var(--hp-glow)_0%,transparent_70%)] opacity-20 blur-3xl"
        />
        <div
          aria-hidden="true"
          className="pointer-events-none absolute bottom-0 -left-24 h-[320px] w-[320px] rounded-full bg-[radial-gradient(circle,var(--hp-glow-deep)_0%,transparent_70%)] opacity-15 blur-3xl"
        />

        <div className="relative z-10 mx-auto grid max-w-6xl gap-14 px-5 pt-16 pb-20 sm:pt-24 sm:pb-24 lg:grid-cols-[1.05fr_0.95fr] lg:items-center lg:pb-28">
          <div className="hp-rise">
            <p className="hp-font-mono text-[11px] tracking-[0.35em] text-(--hp-glow) uppercase">
              Sistem Kasir &amp; Self-Order QRIS
            </p>
            <h1 className="hp-font-display mt-5 text-[2.6rem] leading-[1.05] font-bold tracking-tight text-(--hp-dark-ink) sm:text-6xl">
              Setiap meja,
              <br />
              kasir sendiri.
            </h1>
            <p className="mt-6 max-w-md text-[16px] leading-relaxed text-(--hp-dark-ink-soft)">
              Pelanggan pesan &amp; bayar QRIS langsung dari HP di meja. Begitu lunas, pesanan
              otomatis masuk ke kasir — pemilik memantau semua transaksi dari satu dasbor,
              real-time.
            </p>
            <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center">
              <a
                href="#konsultasi"
                className="inline-flex items-center justify-center gap-1.5 rounded-full bg-[linear-gradient(120deg,var(--hp-glow),var(--hp-glow-deep))] px-6 py-3 text-sm font-semibold text-(--hp-dark) transition-transform hover:scale-[1.02] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-(--hp-glow)"
              >
                Konsultasi Harga
                <ChevronRight className="h-4 w-4" aria-hidden="true" />
              </a>
              <a
                href="#cara-kerja"
                className="inline-flex items-center justify-center rounded-full border border-white/15 px-6 py-3 text-sm font-semibold text-(--hp-dark-ink) transition-colors hover:border-white/30 hover:bg-white/5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-(--hp-glow)"
              >
                Cara Kerjanya
              </a>
            </div>
            <p className="mt-8 flex items-center gap-2 hp-font-mono text-[11px] tracking-[0.15em] text-(--hp-dark-ink-soft) uppercase">
              <ShieldCheck className="h-3.5 w-3.5 text-(--hp-glow)" aria-hidden="true" />
              Pembayaran QRIS diproses aman melalui Tripay
            </p>
          </div>

          <div className="flex flex-col gap-4">
            {TICKETS.map((t, i) => {
              const status = STATUS_STYLE[t.status];
              const style: TicketStyle = {
                "--hp-delay": `${i * 170 + 140}ms`,
                "--hp-tilt": t.tilt,
              };
              return (
                <div
                  key={t.table}
                  style={style}
                  className="hp-ticket rounded-2xl border border-white/10 bg-(--hp-dark-raised) p-5 shadow-[0_20px_40px_-24px_rgba(0,0,0,0.6)]"
                >
                  <div className="flex items-center justify-between">
                    <span className="flex items-center gap-2 hp-font-mono text-[12px] font-medium tracking-[0.1em] text-(--hp-dark-ink) uppercase">
                      {t.status === "baru" && (
                        <span
                          className="hp-pulse-dot h-1.5 w-1.5 rounded-full bg-(--hp-amber)"
                          aria-hidden="true"
                        />
                      )}
                      {t.table}
                    </span>
                    <span
                      className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 hp-font-mono text-[10px] font-medium tracking-[0.08em] uppercase ${status.classes}`}
                    >
                      {t.status === "lunas" && <Check className="h-3 w-3" aria-hidden="true" />}
                      {status.label}
                    </span>
                  </div>
                  <p className="mt-3 text-[13px] leading-relaxed text-(--hp-dark-ink-soft)">
                    {t.items}
                  </p>
                  <p className="hp-glow-num mt-2 text-lg font-medium text-(--hp-dark-ink)">
                    {t.price}
                  </p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* ── Fitur — presented as status panels, the same visual language as the till. ── */}
      <section id="produk" className="px-5 py-20 sm:py-24">
        <div className="mx-auto max-w-6xl">
          <p className="hp-font-mono text-[11px] tracking-[0.3em] text-(--hp-glow-deep) uppercase">
            Yang Anda Dapatkan
          </p>
          <h2 className="hp-font-display mt-3 max-w-xl text-2xl font-bold tracking-tight text-(--hp-ink) sm:text-4xl">
            Satu sistem, empat pekerjaan yang biasanya terpisah
          </h2>

          <div className="mt-10 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            {FEATURES.map((f) => (
              <div
                key={f.name}
                className="rounded-2xl border border-(--hp-line) bg-(--hp-surface) p-6 transition-shadow hover:shadow-[0_16px_40px_-28px_rgba(20,40,35,0.35)]"
              >
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-[linear-gradient(135deg,var(--hp-glow),var(--hp-glow-deep))]">
                  <f.icon className="h-5 w-5 text-(--hp-dark)" aria-hidden="true" />
                </div>
                <p className="hp-font-display mt-4 text-[15px] font-bold text-(--hp-ink)">
                  {f.name}
                </p>
                <p className="mt-1.5 text-[13px] leading-relaxed text-(--hp-ink-soft)">
                  {f.detail}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <div className="hp-rule-fade mx-auto max-w-6xl px-5" aria-hidden="true" />

      {/* ── Cara kerja — a genuine sequence, so the connected numbering is earned. ── */}
      <section id="cara-kerja" className="px-5 py-20 sm:py-24">
        <div className="mx-auto max-w-6xl">
          <p className="text-center hp-font-mono text-[11px] tracking-[0.3em] text-(--hp-glow-deep) uppercase">
            Cara Kerja Self-Order
          </p>
          <h2 className="hp-font-display mx-auto mt-3 max-w-xl text-center text-2xl font-bold tracking-tight text-(--hp-ink) sm:text-4xl">
            Dari scan meja sampai pesanan diproses
          </h2>

          <ol className="mt-12 grid gap-x-4 gap-y-10 sm:grid-cols-2 lg:flex lg:items-start lg:gap-0">
            {STEPS.flatMap((s, i) => {
              const card = (
                <li
                  key={s.n}
                  className="flex flex-col items-start lg:flex-1 lg:items-center lg:text-center"
                >
                  <span className="hp-font-display flex h-11 w-11 shrink-0 items-center justify-center rounded-full border-2 border-(--hp-glow) font-bold text-(--hp-ink)">
                    {s.n}
                  </span>
                  <p className="hp-font-display mt-3 text-[15px] font-bold text-(--hp-ink)">
                    {s.title}
                  </p>
                  <p className="mt-1.5 max-w-[26ch] text-[13px] leading-relaxed text-(--hp-ink-soft)">
                    {s.detail}
                  </p>
                </li>
              );
              if (i === STEPS.length - 1) return [card];
              const connector = (
                <li
                  key={`${s.n}-connector`}
                  aria-hidden="true"
                  className="hidden shrink-0 items-center lg:flex lg:w-10 lg:translate-y-[22px]"
                >
                  <span className="h-px w-full bg-[linear-gradient(to_right,var(--hp-glow),var(--hp-line))]" />
                  <ChevronRight className="h-4 w-4 shrink-0 text-(--hp-glow-deep)" />
                </li>
              );
              return [card, connector];
            })}
          </ol>
        </div>
      </section>

      {/* ── Harga — enterprise: no self-serve price tags, one consultative path to sales. ── */}
      <section id="konsultasi" className="bg-(--hp-surface) px-5 py-20 sm:py-24">
        <div className="mx-auto max-w-5xl">
          <p className="text-center hp-font-mono text-[11px] tracking-[0.3em] text-(--hp-glow-deep) uppercase">
            Harga &amp; Kemitraan
          </p>
          <h2 className="hp-font-display mx-auto mt-3 max-w-xl text-center text-2xl font-bold tracking-tight text-(--hp-ink) sm:text-4xl">
            Skema harga disesuaikan dengan skala bisnis Anda
          </h2>
          <p className="mx-auto mt-4 max-w-lg text-center text-[15px] leading-relaxed text-(--hp-ink-soft)">
            Setiap toko, kafe, dan restoran punya kebutuhan berbeda — jumlah meja, staf, dan volume
            transaksi. Tim kami akan membantu menyusun skema kerja sama yang tepat, langsung dari
            konsultasi awal.
          </p>

          <div className="mt-12 grid gap-10 rounded-2xl border border-(--hp-line) bg-(--hp-surface) px-8 py-10 sm:px-12 sm:py-12 lg:grid-cols-[1.1fr_0.9fr] lg:items-center lg:gap-14">
            <div>
              <p className="hp-font-display text-lg font-bold text-(--hp-ink)">
                Yang Anda dapatkan dari konsultasi
              </p>
              <ul className="mt-5 space-y-3.5">
                {CONSULT_POINTS.map((point) => (
                  <li
                    key={point}
                    className="flex items-start gap-3 text-[14px] leading-relaxed text-(--hp-ink-soft)"
                  >
                    <CheckCircle2
                      className="mt-0.5 h-4 w-4 shrink-0 text-(--hp-glow-deep)"
                      aria-hidden="true"
                    />
                    {point}
                  </li>
                ))}
              </ul>
            </div>

            <div className="hp-dark-section relative overflow-hidden rounded-2xl px-7 py-9 text-center">
              <div className="relative z-10">
                <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-[linear-gradient(135deg,var(--hp-glow),var(--hp-glow-deep))]">
                  <Mail className="h-6 w-6 text-(--hp-dark)" aria-hidden="true" />
                </div>
                <p className="mt-4 text-[13px] text-(--hp-dark-ink-soft)">
                  Konsultasi &amp; penawaran harga
                </p>
                <a
                  href="mailto:cs@elcodelabs.com"
                  className="hp-font-display mt-1 block text-xl font-bold text-(--hp-dark-ink) transition-colors hover:text-(--hp-glow)"
                >
                  cs@elcodelabs.com
                </a>
                <a
                  href="mailto:cs@elcodelabs.com"
                  className="mt-6 inline-flex items-center justify-center gap-1.5 rounded-full bg-[linear-gradient(120deg,var(--hp-glow),var(--hp-glow-deep))] px-6 py-3 text-sm font-semibold text-(--hp-dark) transition-transform hover:scale-[1.02] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-(--hp-glow)"
                >
                  Kirim Email ke Tim Kami
                  <ChevronRight className="h-4 w-4" aria-hidden="true" />
                </a>
                <p className="mt-4 hp-font-mono text-[10px] tracking-[0.15em] text-(--hp-dark-ink-soft) uppercase">
                  Respon dalam 1×24 jam kerja
                </p>
              </div>
            </div>
          </div>

          <p className="mt-8 text-center text-[13px] text-(--hp-ink-soft)">
            Lihat juga{" "}
            <Link
              to={ROUTE_PATHS.homepageTerms}
              className="font-medium text-(--hp-glow-deep) underline underline-offset-2"
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
