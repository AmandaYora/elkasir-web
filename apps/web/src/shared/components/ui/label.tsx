import { forwardRef } from "react";
import { cn } from "@/shared/lib/cn";

export type LabelProps = React.LabelHTMLAttributes<HTMLLabelElement>;

export const Label = forwardRef<HTMLLabelElement, LabelProps>(({ className, ...props }, ref) => (
  <label ref={ref} className={cn("text-sm font-medium text-text", className)} {...props} />
));
Label.displayName = "Label";
