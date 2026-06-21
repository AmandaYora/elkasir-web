import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { cn } from "@/shared/lib/cn";

export interface DropdownProps {
  trigger: React.ReactNode;
  children: React.ReactNode; // DropdownItem list
  align?: "start" | "end";
  className?: string;
}

// Minimal dropdown menu (button trigger + click-outside close). No headless deps.
// The menu is rendered in a portal with fixed positioning so it is never clipped by
// an ancestor's `overflow-hidden` (e.g. a Card wrapping a Table).
export function Dropdown({ trigger, children, align = "end", className }: DropdownProps) {
  const [open, setOpen] = useState(false);
  const [style, setStyle] = useState<React.CSSProperties>({});
  const triggerRef = useRef<HTMLSpanElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  // Position the portaled menu against the trigger's viewport rect.
  useLayoutEffect(() => {
    if (!open) return;
    const place = () => {
      const t = triggerRef.current?.getBoundingClientRect();
      if (!t) return;
      const gap = 4;
      const menuH = menuRef.current?.offsetHeight ?? 0;
      const below = t.bottom + gap;
      const flipUp = menuH > 0 && below + menuH > window.innerHeight && t.top - gap - menuH > 0;
      const next: React.CSSProperties = {
        position: "fixed",
        top: flipUp ? t.top - gap - menuH : below,
        opacity: 1,
      };
      if (align === "end") next.right = Math.max(8, window.innerWidth - t.right);
      else next.left = t.left;
      setStyle(next);
    };
    place();
    window.addEventListener("resize", place);
    window.addEventListener("scroll", place, true);
    return () => {
      window.removeEventListener("resize", place);
      window.removeEventListener("scroll", place, true);
    };
  }, [open, align]);

  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      const target = e.target as Node;
      if (triggerRef.current?.contains(target) || menuRef.current?.contains(target)) return;
      setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div className="relative inline-block">
      <span ref={triggerRef} onClick={() => setOpen((v) => !v)}>
        {trigger}
      </span>
      {open &&
        createPortal(
          <div
            ref={menuRef}
            style={{ position: "fixed", opacity: 0, ...style }}
            className={cn(
              "z-50 min-w-40 overflow-hidden rounded-md border border-border bg-surface p-1 shadow-lg",
              className,
            )}
            onClick={() => setOpen(false)}
          >
            {children}
          </div>,
          document.body,
        )}
    </div>
  );
}

export interface DropdownItemProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  danger?: boolean;
}

export function DropdownItem({ className, danger, ...props }: DropdownItemProps) {
  return (
    <button
      className={cn(
        "flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm transition-colors hover:bg-surface-muted",
        danger ? "text-danger hover:bg-danger-soft" : "text-text",
        className,
      )}
      {...props}
    />
  );
}
