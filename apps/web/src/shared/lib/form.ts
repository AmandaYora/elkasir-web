import type { ZodError } from "zod";

/**
 * Map a Zod safeParse failure to the first error message per field, for inline
 * field-level display (NN/g: show the error next to the offending field, not only a toast).
 */
export function zodFieldErrors(error: ZodError): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [key, msgs] of Object.entries(error.flatten().fieldErrors)) {
    if (msgs && msgs.length) out[key] = msgs[0] as string;
  }
  return out;
}
