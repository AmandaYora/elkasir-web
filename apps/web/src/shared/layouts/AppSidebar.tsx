import { NavLink } from "react-router-dom";
import { appBrand } from "@/shared/constants/brand";
import { cn } from "@/shared/lib/cn";

export type NavItem = {
  title: string;
  to: string;
  icon: React.ComponentType<{ className?: string }>;
  /** NavLink `end` — only needed for a group's own root path (e.g. the dashboard). */
  end?: boolean;
};
export type NavGroup = { label: string; items: NavItem[] };

// Domain-agnostic dashboard sidebar — takes its nav sections and subtitle as props so it can be
// reused for both the tenant admin dashboard and Konsol Platform (§2.2/§2.12), with no internal
// knowledge of either domain's routes or auth store.
export function AppSidebar({
  groups,
  subtitle = "Admin POS",
}: {
  groups: NavGroup[];
  subtitle?: string;
}) {
  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-border bg-surface">
      <div className="flex items-center gap-2.5 border-b border-border px-4 py-4">
        <img src="/elkasir-logo.png" alt={appBrand} className="h-9 w-9 shrink-0" />
        <div className="flex min-w-0 flex-col">
          <span className="truncate text-sm font-semibold leading-tight text-text">{appBrand}</span>
          <span className="truncate text-[11px] text-muted">{subtitle}</span>
        </div>
      </div>

      <nav className="flex-1 overflow-y-auto py-2">
        {groups.map((group) => (
          <div key={group.label} className="px-2 py-1.5">
            <p className="px-3 pb-1 text-[11px] font-medium uppercase tracking-wider text-muted">
              {group.label}
            </p>
            <ul className="space-y-0.5">
              {group.items.map((item) => (
                <li key={item.title}>
                  <NavLink
                    to={item.to}
                    end={item.end}
                    className={({ isActive }: { isActive: boolean }) =>
                      cn(
                        "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors",
                        isActive
                          ? "bg-primary-soft font-medium text-primary"
                          : "text-text hover:bg-surface-muted",
                      )
                    }
                  >
                    <item.icon className="h-4 w-4 shrink-0" />
                    <span>{item.title}</span>
                  </NavLink>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </nav>
    </aside>
  );
}
