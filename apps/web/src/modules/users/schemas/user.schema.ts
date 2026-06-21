import { z } from "zod";

const base = {
  name: z.string().min(1, "Nama wajib diisi"),
  email: z.string().min(1, "Email wajib diisi").email("Email tidak valid"),
  role: z.enum(["owner", "admin", "manager", "viewer"]),
  status: z.enum(["active", "inactive"]),
};

export const adminCreateSchema = z.object({
  ...base,
  password: z.string().min(1, "Password wajib diisi"),
});

export const adminUpdateSchema = z.object(base);

export type AdminCreateValues = z.infer<typeof adminCreateSchema>;
export type AdminUpdateValues = z.infer<typeof adminUpdateSchema>;
