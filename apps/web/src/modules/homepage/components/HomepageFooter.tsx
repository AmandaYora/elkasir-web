import { MapPin, Phone } from "lucide-react";
import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export function HomepageFooter() {
  return (
    <footer className="w-full">
      <div className="relative z-10 mx-auto max-w-7xl px-6 py-16">
        <div className="grid gap-10 sm:grid-cols-3">
          <div>
            <div className="flex items-center gap-3">
              <img src="/elkasir-logo.png" alt="Elkasir Logo" className="h-8 w-8 rounded-lg shadow-sm" />
              <span className="hp-font-display text-base font-bold tracking-tight text-(--hp-ink)">
                Elkasir
              </span>
            </div>
            <p className="mt-2 hp-font-mono text-[10px] font-semibold tracking-[0.2em] text-(--hp-ink-soft) uppercase">
              Dikembangkan oleh Elcodelabs
            </p>
            <p className="mt-4 max-w-[30ch] text-sm leading-relaxed text-(--hp-ink-soft)">
              Sistem kasir &amp; self-order QRIS untuk warung, kafe, dan restoran.
            </p>
            <div className="mt-6 space-y-3 text-sm text-(--hp-ink-soft)">
              <p className="flex items-start gap-2.5">
                <MapPin className="mt-0.5 h-4 w-4 shrink-0 text-(--hp-glow-deep)" aria-hidden="true" />
                <span className="max-w-[30ch] leading-relaxed">
                  Gland Ciwastra Park, Jl. Morinda X No. 26, Bojongsoang, Kab. Bandung
                </span>
              </p>
              <p className="flex items-center gap-2.5">
                <Phone className="h-4 w-4 shrink-0 text-(--hp-glow-deep)" aria-hidden="true" />
                <a href="tel:+6285173471146" className="font-medium transition-colors hover:text-(--hp-glow-deep)">
                  0851-7347-1146
                </a>
              </p>
            </div>
          </div>
          <div>
            <p className="hp-font-mono text-[11px] font-semibold uppercase tracking-[0.2em] text-(--hp-ink-soft)">
              Produk
            </p>
            <ul className="mt-4 space-y-3 text-sm text-(--hp-ink)">
              <li>
                <Link
                  to={ROUTE_PATHS.homepage}
                  className="font-medium transition-colors hover:text-(--hp-glow-deep)"
                >
                  Beranda
                </Link>
              </li>
              <li>
                <Link to={ROUTE_PATHS.login} className="font-medium transition-colors hover:text-(--hp-glow-deep)">
                  Masuk ke Dasbor
                </Link>
              </li>
            </ul>
          </div>
          <div>
            <p className="hp-font-mono text-[11px] font-semibold uppercase tracking-[0.2em] text-(--hp-ink-soft)">
              Bantuan
            </p>
            <ul className="mt-4 space-y-3 text-sm text-(--hp-ink)">
              <li>
                <Link
                  to={ROUTE_PATHS.homepageContact}
                  className="font-medium transition-colors hover:text-(--hp-glow-deep)"
                >
                  Kontak Customer Service
                </Link>
              </li>
              <li>
                <Link
                  to={ROUTE_PATHS.homepageTerms}
                  className="font-medium transition-colors hover:text-(--hp-glow-deep)"
                >
                  Syarat &amp; Ketentuan
                </Link>
              </li>
            </ul>
          </div>
        </div>

        <div className="mt-12 h-px w-full bg-gradient-to-r from-transparent via-(--hp-line) to-transparent" aria-hidden="true" />

        <div className="mt-8 flex flex-col gap-2 hp-font-mono text-xs font-medium text-(--hp-ink-soft) sm:flex-row sm:items-center sm:justify-between">
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
