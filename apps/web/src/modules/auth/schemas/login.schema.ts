import { z } from "zod";

// Email here may be a plain username (e.g. "admin"), so keep it lenient.
export const loginSchema = z.object({
  email: z.string().min(1, "Email wajib diisi"),
  password: z.string().min(1, "Password wajib diisi"),
});

export type LoginFormValues = z.infer<typeof loginSchema>;
