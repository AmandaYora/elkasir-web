import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export function HomepageNav() {
  return (
    <header className="sticky top-0 z-30 border-b border-(--hp-line) bg-white">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
        <Link to={ROUTE_PATHS.homepage} className="flex items-center gap-3">
          <img src="/elkasir-logo.png" alt="Elkasir Logo" className="h-8 w-8 rounded" />
          <span className="hp-font-display text-lg font-bold tracking-tight text-(--hp-ink)">
            Elkasir
          </span>
        </Link>

        <nav className="hidden md:flex items-center gap-8 text-sm font-medium text-(--hp-ink-soft)">
          <Link to={ROUTE_PATHS.homepage} className="transition-colors hover:text-(--hp-primary)">
            Beranda
          </Link>
          <Link
            to={ROUTE_PATHS.homepageTerms}
            className="transition-colors hover:text-(--hp-primary)"
          >
            Syarat &amp; Ketentuan
          </Link>
          <Link
            to={ROUTE_PATHS.homepageContact}
            className="transition-colors hover:text-(--hp-primary)"
          >
            Kontak
          </Link>
        </nav>

        <div className="flex items-center gap-4">
          <Link
            to={ROUTE_PATHS.login}
            className="whitespace-nowrap rounded-md bg-(--hp-primary) px-5 py-2 text-sm font-medium text-white transition-colors hover:bg-(--hp-primary-hover) focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-(--hp-primary)"
          >
            Masuk
          </Link>
        </div>
      </div>
    </header>
  );
}
