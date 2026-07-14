import { MapPin, Phone } from "lucide-react";
import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export function HomepageFooter() {
  return (
    <footer className="hp-dark-section">
      <div className="relative z-10 mx-auto max-w-6xl px-5 py-14">
        <div className="grid gap-10 sm:grid-cols-3">
          <div>
            <div className="flex items-center gap-2.5">
              <img src="/elkasir-logo.png" alt="" className="h-7 w-7 rounded-lg" />
              <span className="hp-font-display text-sm font-bold tracking-tight text-(--hp-dark-ink)">
                Elkasir
              </span>
            </div>
            <p className="mt-1 hp-font-mono text-[10px] tracking-[0.2em] text-(--hp-dark-ink-soft) uppercase">
              by Elcodelabs
            </p>
            <p className="mt-3 max-w-[30ch] text-sm leading-relaxed text-(--hp-dark-ink-soft)">
              Sistem kasir &amp; self-order QRIS untuk warung, kafe, dan restoran.
            </p>
            <div className="mt-5 space-y-2.5 text-sm text-(--hp-dark-ink-soft)">
              <p className="flex items-start gap-2.5">
                <MapPin className="mt-0.5 h-4 w-4 shrink-0 text-(--hp-glow)" aria-hidden="true" />
                <span className="max-w-[30ch] leading-relaxed">
                  Gland Ciwastra Park, Jl. Morinda X No. 26, Bojongsoang, Kab. Bandung
                </span>
              </p>
              <p className="flex items-center gap-2.5">
                <Phone className="h-4 w-4 shrink-0 text-(--hp-glow)" aria-hidden="true" />
                <a href="tel:+6285173471146" className="transition-colors hover:text-(--hp-glow)">
                  0851-7347-1146
                </a>
              </p>
            </div>
          </div>
          <div>
            <p className="hp-font-mono text-[11px] font-medium uppercase tracking-[0.2em] text-(--hp-dark-ink-soft)">
              Produk
            </p>
            <ul className="mt-3 space-y-2 text-sm text-(--hp-dark-ink)">
              <li>
                <Link
                  to={ROUTE_PATHS.homepage}
                  className="transition-colors hover:text-(--hp-glow)"
                >
                  Beranda
                </Link>
              </li>
              <li>
                <Link to={ROUTE_PATHS.login} className="transition-colors hover:text-(--hp-glow)">
                  Masuk ke Dasbor
                </Link>
              </li>
            </ul>
          </div>
          <div>
            <p className="hp-font-mono text-[11px] font-medium uppercase tracking-[0.2em] text-(--hp-dark-ink-soft)">
              Bantuan
            </p>
            <ul className="mt-3 space-y-2 text-sm text-(--hp-dark-ink)">
              <li>
                <Link
                  to={ROUTE_PATHS.homepageContact}
                  className="transition-colors hover:text-(--hp-glow)"
                >
                  Kontak Customer Service
                </Link>
              </li>
              <li>
                <Link
                  to={ROUTE_PATHS.homepageTerms}
                  className="transition-colors hover:text-(--hp-glow)"
                >
                  Syarat &amp; Ketentuan
                </Link>
              </li>
            </ul>
          </div>
        </div>

        <div className="hp-rule-fade-dark mt-10" aria-hidden="true" />

        <div className="mt-6 flex flex-col gap-1.5 hp-font-mono text-[11px] text-(--hp-dark-ink-soft) sm:flex-row sm:items-center sm:justify-between">
          <span>
            © {new Date().getFullYear()} Elkasir by Elcodelabs — PT. Prayora Karya Pratama. Seluruh
            hak cipta dilindungi.
          </span>
          <span>elkasir.elcodelabs.com</span>
        </div>
      </div>
    </footer>
  );
}
