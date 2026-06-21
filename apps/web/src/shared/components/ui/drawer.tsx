import { useEffect } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { cn } from "@/shared/lib/cn";

export interface DrawerProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  children?: React.ReactNode;
  side?: "right" | "left";
  className?: string;
}

// Minimal side drawer (overlay + sliding panel, Esc to close).
export function Drawer({
  open,
  onClose,
  title,
  description,
  children,
  side = "right",
  className,
}: DrawerProps) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = "";
    };
  }, [open, onClose]);

  if (!open) return null;

  return createPortal(
    <div className="fixed inset-0 z-50">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        className={cn(
          "absolute top-0 z-10 flex h-full w-full max-w-md flex-col border-border bg-surface shadow-xl",
          side === "right" ? "right-0 border-l" : "left-0 border-r",
          className,
        )}
      >
        <div className="flex items-start justify-between border-b border-border p-4">
          <div className="flex flex-col gap-1">
            {title && <h2 className="text-base font-semibold text-text">{title}</h2>}
            {description && <p className="text-sm text-muted">{description}</p>}
          </div>
          <button
            onClick={onClose}
            className="rounded-md p-1 text-muted transition-colors hover:bg-surface-muted"
            aria-label="Tutup"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-4">{children}</div>
      </div>
    </div>,
    document.body,
  );
}
