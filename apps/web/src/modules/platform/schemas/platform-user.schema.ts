import { z } from "zod";

export const createPlatformUserSchema = z.object({
  name: z.string().min(1, "Nama wajib diisi"),
  email: z.string().min(1, "Email wajib diisi").email("Email tidak valid"),
  password: z.string().min(6, "Password minimal 6 karakter"),
});

export const resetPlatformUserPasswordSchema = z.object({
  password: z.string().min(6, "Password minimal 6 karakter"),
});

export type CreatePlatformUserFormValues = z.infer<typeof createPlatformUserSchema>;
