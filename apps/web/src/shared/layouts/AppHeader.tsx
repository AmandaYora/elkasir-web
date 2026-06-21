import { useNavigate } from "react-router-dom";
import { LogOut } from "lucide-react";
import { useAuthStore } from "@/shared/stores/auth.store";
import { adminRoleLabel } from "@/shared/constants/brand";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Dropdown, DropdownItem } from "@/shared/components/ui/dropdown";

export function AppHeader({ title }: { title?: string }) {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();

  const initials = user
    ? user.name
        .split(" ")
        .map((w) => w[0])
        .slice(0, 2)
        .join("")
    : "?";

  const onLogout = async () => {
    await logout();
    navigate(ROUTE_PATHS.login, { replace: true });
  };

  return (
    <header className="flex h-14 shrink-0 items-center justify-between border-b border-border bg-surface px-4 md:px-6">
      <h1 className="text-sm font-semibold text-text">{title}</h1>
      <Dropdown
        trigger={
          <button className="flex items-center gap-2.5 rounded-lg p-1 transition-colors hover:bg-surface-muted">
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary-soft text-xs font-medium text-primary">
              {initials}
            </span>
            <span className="hidden flex-col items-start text-left sm:flex">
              <span className="text-xs font-medium text-text">{user?.name ?? "—"}</span>
              <span className="text-[11px] text-muted">
                {user ? (adminRoleLabel[user.role] ?? user.role) : ""}
              </span>
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
