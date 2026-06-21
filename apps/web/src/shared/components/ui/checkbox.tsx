import { forwardRef } from "react";
import { cn } from "@/shared/lib/cn";

export type CheckboxProps = React.InputHTMLAttributes<HTMLInputElement>;

export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(({ className, ...props }, ref) => (
  <input
    ref={ref}
    type="checkbox"
    className={cn(
      "h-4 w-4 rounded border-border text-primary accent-primary focus-visible:ring-2 focus-visible:ring-primary/30",
      className,
    )}
    {...props}
  />
));
Checkbox.displayName = "Checkbox";
