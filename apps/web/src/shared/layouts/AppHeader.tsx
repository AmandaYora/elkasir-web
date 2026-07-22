import { LogOut, Menu } from "lucide-react";
import { Dropdown, DropdownItem } from "@/shared/components/ui/dropdown";

export interface HeaderUser {
  name: string;
  roleLabel: string;
}

// Domain-agnostic dashboard header — takes the current user + a logout handler as props (no
// internal useAuthStore/useNavigate) so it can be reused for both the tenant admin dashboard
// and Konsol Platform (§2.2), each supplying their own session store. `onMenuClick` opens the
// off-canvas AppSidebar drawer on mobile — omit it (e.g. on auth screens with no sidebar) to
// hide the hamburger button entirely.
export function AppHeader({
  title,
  user,
  onLogout,
  onMenuClick,
}: {
  title?: string;
  user: HeaderUser | null;
  onLogout: () => void;
  onMenuClick?: () => void;
}) {
  const initials = user
    ? user.name
        .split(" ")
        .map((w) => w[0])
        .slice(0, 2)
        .join("")
    : "?";

  return (
    <header className="flex h-14 shrink-0 items-center justify-between gap-2 border-b border-border bg-surface px-4 md:px-6">
      <div className="flex min-w-0 items-center gap-2">
        {onMenuClick && (
          <button
            onClick={onMenuClick}
            className="-ml-1 rounded-md p-1.5 text-muted transition-colors hover:bg-surface-muted md:hidden"
            aria-label="Buka menu"
          >
            <Menu className="h-5 w-5" />
          </button>
        )}
        <h1 className="truncate text-sm font-semibold text-text">{title}</h1>
      </div>
      <Dropdown
        trigger={
          <button className="flex items-center gap-2.5 rounded-lg p-1 transition-colors hover:bg-surface-muted">
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary-soft text-xs font-medium text-primary">
              {initials}
            </span>
            <span className="hidden flex-col items-start text-left sm:flex">
              <span className="text-xs font-medium text-text">{user?.name ?? "—"}</span>
              <span className="text-[11px] text-muted">{user?.roleLabel ?? ""}</span>
            </span>
          </button>
        }
      >
        <DropdownItem danger onClick={onLogout}>
          <LogOut className="h-3.5 w-3.5" /> Keluar
        </DropdownItem>
      </Dropdown>
    </header>
  );
}
