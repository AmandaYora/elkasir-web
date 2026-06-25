import type { InputHTMLAttributes } from "react";
import { Input } from "@/shared/components/ui/input";

const format = (n: number) => (n ? n.toLocaleString("id-ID") : "");
const parse = (s: string) => Number(s.replace(/\D/g, "")) || 0;

type Props = Omit<InputHTMLAttributes<HTMLInputElement>, "value" | "onChange" | "type"> & {
  value: number;
  onChange: (n: number) => void;
};

/**
 * Rupiah amount input that displays thousand separators (e.g. "200.000") while keeping a
 * plain numeric value in state — an input mask that prevents mis-reads/mis-typing of money
 * fields for non-technical users (NN/g: input masks reduce errors and ease double-checking).
 */
export function MoneyInput({ value, onChange, ...props }: Props) {
  return (
    <Input
      inputMode="numeric"
      value={format(value)}
      onChange={(e) => onChange(parse(e.target.value))}
      {...props}
    />
  );
}
