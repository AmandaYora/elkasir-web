import { Button } from "./button";

export interface PaginationProps {
  page: number;
  totalPages: number;
  total?: number;
  onPageChange: (page: number) => void;
  label?: string;
}

// Simple prev/next pager used by list pages.
export function Pagination({ page, totalPages, total, onPageChange, label }: PaginationProps) {
  return (
    <div className="flex items-center justify-between gap-2 border-t border-border px-4 py-3 text-sm">
      <span className="text-muted">{label ?? (total != null ? `${total} data` : "")}</span>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
        >
          Sebelumnya
        </Button>
        <span className="text-xs text-muted">
          Halaman {page} dari {Math.max(1, totalPages)}
        </span>
        <Button
          variant="outline"
          size="sm"
          disabled={page >= totalPages}
          onClick={() => onPageChange(page + 1)}
        >
          Berikutnya
        </Button>
      </div>
    </div>
  );
}
