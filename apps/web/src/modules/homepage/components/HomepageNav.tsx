import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export function HomepageNav() {
  return (
    <header className="hp-dark-section sticky top-0 z-30 border-b border-white/10 bg-(--hp-dark)/95 backdrop-blur">
      <div className="relative z-10 mx-auto flex max-w-6xl items-center justify-between px-5 py-3.5">
        <Link to={ROUTE_PATHS.homepage} className="flex items-center gap-2.5">
          <img src="/elkasir-logo.png" alt="" className="h-8 w-8 rounded-lg" />
          <span className="hp-font-display text-[15px] font-bold tracking-tight text-(--hp-dark-ink)">
            Elkasir
          </span>
        </Link>
        <nav className="flex items-center gap-6 text-sm font-medium text-(--hp-dark-ink-soft)">
          <Link
            to={ROUTE_PATHS.homepageTerms}
            className="hidden transition-colors hover:text-(--hp-dark-ink) sm:inline"
          >
            Syarat &amp; Ketentuan
          </Link>
          <Link
            to={ROUTE_PATHS.homepageContact}
            className="hidden transition-colors hover:text-(--hp-dark-ink) sm:inline"
          >
            Kontak
          </Link>
          <Link
            to={ROUTE_PATHS.login}
            className="whitespace-nowrap rounded-full border border-(--hp-glow)/40 px-4 py-2 font-semibold text-(--hp-dark-ink) transition-colors hover:border-(--hp-glow) hover:bg-(--hp-glow)/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-(--hp-glow)"
          >
            Masuk
          </Link>
        </nav>
      </div>
    </header>
  );
}
