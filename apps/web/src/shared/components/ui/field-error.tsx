import { AlertCircle } from "lucide-react";

/**
 * Inline field-level error shown directly under the offending input. Uses an icon + text
 * (redundant, non-color cues) so the signal is not conveyed by color alone — per NN/g /
 * WCAG 1.4.1 form-error guidance.
 */
export function FieldError({ msg }: { msg?: string }) {
  if (!msg) return null;
  return (
    <p className="flex items-center gap-1 text-xs font-medium text-danger" role="alert">
      <AlertCircle className="h-3.5 w-3.5 shrink-0" />
      <span>{msg}</span>
    </p>
  );
}
