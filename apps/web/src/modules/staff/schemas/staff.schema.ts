import { z } from "zod";

// Base fields shared by create & update.
const base = {
  name: z.string().min(1, "Nama wajib diisi"),
  username: z.string().min(1, "Username wajib diisi"),
  email: z.string().email("Email tidak valid").optional(),
  role: z.enum(["cashier", "supervisor"]),
  status: z.enum(["active", "inactive"]),
};

export const staffCreateSchema = z.object({
  ...base,
  password: z.string().min(1, "Password wajib diisi"),
});

export const staffUpdateSchema = z.object(base);

export type StaffCreateValues = z.infer<typeof staffCreateSchema>;
export type StaffUpdateValues = z.infer<typeof staffUpdateSchema>;
