import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { HomepageNav } from "@/modules/homepage/components/HomepageNav";
import { HomepageFooter } from "@/modules/homepage/components/HomepageFooter";
import "@/modules/homepage/styles.css";

const SECTIONS: { title: string; body: string[] }[] = [
  {
    title: "1. Tentang Layanan",
    body: [
      "Elkasir adalah layanan perangkat lunak (Software-as-a-Service) berupa sistem Point-of-Sale (POS) untuk usaha food & beverage — meliputi manajemen menu, transaksi kasir, self-order QRIS bagi pelanggan, manajemen shift, dan laporan penjualan. Elkasir dioperasikan oleh Elcodelabs.",
      'Dengan mendaftar dan menggunakan Elkasir, pemilik usaha ("Pengguna") menyetujui syarat dan ketentuan berikut.',
    ],
  },
  {
    title: "2. Akun & Langganan",
    body: [
      "Pengguna mendaftarkan tokonya melalui proses onboarding yang disediakan Elkasir, dan bertanggung jawab atas kerahasiaan kredensial akun (email/username dan kata sandi) miliknya sendiri maupun staf yang dibuatkannya akses.",
      "Akses ke fitur Elkasir memerlukan langganan aktif dengan biaya berkala sesuai paket yang dipilih (bulanan atau tahunan, lihat halaman Harga). Langganan diperpanjang secara manual oleh Pengguna sebelum masa aktif berakhir.",
      "Elkasir berhak menangguhkan akses toko yang langganannya tidak aktif, dengan pemberitahuan yang jelas di dalam aplikasi.",
    ],
  },
  {
    title: "3. Pembayaran",
    body: [
      "Pembayaran langganan maupun transaksi self-order QRIS diproses melalui payment gateway pihak ketiga (Tripay), menggunakan metode QRIS. Elkasir tidak menyimpan data kartu atau rekening bank pelanggan.",
      "Seluruh nilai transaksi ditampilkan dalam Rupiah (IDR) dan bersifat final setelah pembayaran dikonfirmasi oleh payment gateway.",
    ],
  },
  {
    title: "4. Tanggung Jawab Pengguna",
    body: [
      "Pengguna bertanggung jawab atas keakuratan data yang dimasukkan ke sistem — termasuk daftar menu, harga, dan informasi toko — karena data tersebut ditampilkan langsung kepada pelanggan akhir Pengguna melalui halaman self-order.",
      "Pengguna dilarang menggunakan Elkasir untuk aktivitas yang melanggar hukum yang berlaku di Indonesia.",
    ],
  },
  {
    title: "5. Ketersediaan Layanan",
    body: [
      "Elkasir berupaya menjaga layanan tetap tersedia, namun tidak menjamin operasional tanpa gangguan 100% sepanjang waktu. Pemeliharaan terjadwal akan diinformasikan sebisa mungkin sebelumnya.",
    ],
  },
  {
    title: "6. Perubahan Ketentuan",
    body: [
      "Elkasir dapat memperbarui syarat dan ketentuan ini dari waktu ke waktu. Perubahan material akan diinformasikan melalui aplikasi atau email terdaftar Pengguna.",
    ],
  },
  {
    title: "7. Hukum yang Berlaku",
    body: ["Syarat dan ketentuan ini diatur dan ditafsirkan berdasarkan hukum Republik Indonesia."],
  },
];

export default function TermsPage() {
  return (
    <div className="hp-root">
      <HomepageNav />

      <section className="mx-auto max-w-2xl px-5 py-16 sm:py-24">
        <p className="hp-font-mono text-[11px] tracking-[0.3em] text-(--hp-glow-deep) uppercase">
          Legal
        </p>
        <h1 className="hp-font-display mt-3 text-3xl font-bold tracking-tight text-(--hp-ink) sm:text-4xl">
          Syarat &amp; Ketentuan
        </h1>
        <p className="hp-font-mono mt-3 text-[11px] text-(--hp-ink-soft)">
          Terakhir diperbarui: 14 Juli 2026
        </p>

        <div className="mt-10 space-y-9">
          {SECTIONS.map((s) => (
            <div key={s.title}>
              <h2 className="hp-font-display text-lg font-bold text-(--hp-ink)">{s.title}</h2>
              <div className="mt-2 space-y-3">
                {s.body.map((p) => (
                  <p key={p} className="text-[15px] leading-relaxed text-(--hp-ink-soft)">
                    {p}
                  </p>
                ))}
              </div>
            </div>
          ))}
        </div>

        <p className="mt-12 border-t border-(--hp-line) pt-6 text-sm text-(--hp-ink-soft)">
          Ada pertanyaan soal ketentuan ini? Hubungi kami di halaman{" "}
          <Link
            to={ROUTE_PATHS.homepageContact}
            className="font-medium text-(--hp-glow-deep) underline underline-offset-2"
          >
            Kontak
          </Link>
          .
        </p>
      </section>

      <HomepageFooter />
    </div>
  );
}
