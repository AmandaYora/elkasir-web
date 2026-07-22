import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { HomepageNav } from "@/modules/homepage/components/HomepageNav";
import { HomepageFooter } from "@/modules/homepage/components/HomepageFooter";
import { FileText, ShieldAlert } from "lucide-react";
import "@/modules/homepage/styles.css";

const SECTIONS: { title: string; body: string[] }[] = [
  {
    title: "1. Tentang Layanan",
    body: [
      "Elkasir adalah sistem manajemen kasir terpadu yang dirancang khusus untuk mendukung operasional bisnis kuliner (warung, kafe, dan restoran). Layanan ini mencakup pengelolaan menu, sistem pemesanan langsung dari meja, pengelolaan shift karyawan, hingga laporan penjualan otomatis.",
      "Dengan mendaftar dan menggunakan Elkasir, Anda sebagai pemilik atau pengelola usaha menyetujui seluruh syarat dan ketentuan layanan kami di bawah ini.",
    ],
  },
  {
    title: "2. Akun & Langganan",
    body: [
      "Anda mendaftarkan toko atau restoran Anda secara resmi melalui proses pendaftaran Elkasir. Anda sepenuhnya bertanggung jawab untuk menjaga kerahasiaan kata sandi akun Anda, termasuk akun staf yang Anda buat di dalam sistem.",
      "Akses penuh ke fitur Elkasir memerlukan status langganan aktif. Biaya langganan dibayarkan secara berkala (bulanan atau tahunan) sesuai dengan paket yang Anda pilih.",
      "Kami berhak menangguhkan sementara akses ke sistem untuk toko yang masa langganannya telah berakhir, dengan pemberitahuan terlebih dahulu di halaman aplikasi.",
    ],
  },
  {
    title: "3. Pembayaran & Transaksi",
    body: [
      "Seluruh transaksi pelanggan yang dilakukan melalui sistem pemesanan meja dan pembayaran biaya langganan diproses secara aman menggunakan standar pembayaran nasional. Kami memastikan perlindungan ketat terhadap privasi dan data transaksi Anda.",
      "Semua nilai transaksi yang tertera dan diselesaikan oleh pelanggan bernilai final dalam mata uang Rupiah.",
    ],
  },
  {
    title: "4. Tanggung Jawab Pengguna",
    body: [
      "Anda bertanggung jawab penuh atas kebenaran data yang Anda masukkan ke dalam sistem, termasuk harga menu, ketersediaan produk, dan nama toko, karena data ini akan dilihat langsung oleh pelanggan Anda.",
      "Sistem Elkasir dilarang keras digunakan untuk segala bentuk aktivitas yang melanggar hukum di Indonesia.",
    ],
  },
  {
    title: "5. Ketersediaan Sistem",
    body: [
      "Kami berkomitmen penuh untuk menjaga kelancaran operasional sistem kasir Anda. Jika ada pemeliharaan sistem yang diperlukan, tim kami akan selalu berupaya memberikan pemberitahuan terlebih dahulu agar operasional bisnis Anda tidak terganggu.",
    ],
  },
  {
    title: "6. Pembaruan Ketentuan",
    body: [
      "Syarat dan ketentuan ini bisa berubah di kemudian hari seiring dengan peningkatan layanan kami. Jika terdapat perubahan penting, kami akan menginformasikannya melalui sistem atau email yang Anda daftarkan.",
    ],
  },
  {
    title: "7. Hukum yang Berlaku",
    body: [
      "Ketentuan ini tunduk dan patuh pada hukum dan peraturan yang berlaku di Republik Indonesia.",
    ],
  },
];

export default function TermsPage() {
  return (
    <div className="hp-root relative min-h-screen flex flex-col bg-(--hp-surface-raised)">
      <HomepageNav />

      <main className="relative z-10 flex-grow px-6 py-24 sm:py-32">
        <div className="mx-auto max-w-3xl hp-animate-fade">
          {/* Header */}
          <div className="mb-16 text-center">
            <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-white border border-(--hp-line) shadow-sm">
              <FileText className="h-8 w-8 text-(--hp-primary)" aria-hidden="true" />
            </div>
            <p className="hp-font-mono text-sm font-semibold tracking-widest text-(--hp-ink-soft) uppercase">
              Informasi Legal
            </p>
            <h1 className="hp-font-display mt-4 text-4xl font-bold tracking-tight text-(--hp-ink) sm:text-5xl">
              Syarat &amp; Ketentuan
            </h1>
            <p className="hp-font-mono mt-6 text-sm text-(--hp-ink-soft) flex items-center justify-center gap-2">
              <ShieldAlert className="h-4 w-4" />
              Terakhir diperbarui: 14 Juli 2026
            </p>
          </div>

          {/* Content inside standard solid Card */}
          <div className="hp-card p-8 sm:p-12">
            <div className="space-y-12">
              {SECTIONS.map((s, index) => (
                <section
                  key={s.title}
                  className={index !== 0 ? "pt-10 border-t border-(--hp-line)" : ""}
                >
                  <h2 className="hp-font-display text-xl font-semibold text-(--hp-ink) tracking-wide">
                    {s.title}
                  </h2>
                  <div className="mt-6 space-y-5">
                    {s.body.map((p, i) => (
                      <p key={i} className="text-sm leading-loose text-(--hp-ink-soft)">
                        {p}
                      </p>
                    ))}
                  </div>
                </section>
              ))}
            </div>
          </div>

          <p className="mt-12 text-center text-sm font-medium text-(--hp-ink-soft)">
            Ada pertanyaan atau kendala? Hubungi tim kami di halaman{" "}
            <Link
              to={ROUTE_PATHS.homepageContact}
              className="text-(--hp-ink) underline underline-offset-4 hover:text-(--hp-primary) transition-colors"
            >
              Kontak
            </Link>
            .
          </p>
        </div>
      </main>

      <div className="relative z-10 border-t border-(--hp-line) bg-white mt-auto">
        <HomepageFooter />
      </div>
    </div>
  );
}
