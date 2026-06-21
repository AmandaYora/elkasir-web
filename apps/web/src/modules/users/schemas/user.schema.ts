import { z } from "zod";

const base = {
  name: z.string().min(1, "Nama wajib diisi"),
  email: z.string().min(1, "Email wajib diisi").email("Email tidak valid"),
  role: z.enum(["owner", "admin", "manager", "viewer"]),
  status: z.enum(["active", "inactive"]),
};

// Username dipakai untuk login (selain email). Selaras dengan validasi backend.
export const usernameSchema = z
  .string()
  .min(3, "Username minimal 3 karakter")
  .max(100, "Username maksimal 100 karakter")
  .regex(
    /^[a-z0-9._-]+$/,
    "Username hanya boleh huruf kecil, angka, titik, garis bawah, atau strip",
  );

// Password baru saat edit (opsional). Kosongkan untuk membiarkan password lama.
export const passwordSchema = z.string().min(6, "Password minimal 6 karakter");

export const adminCreateSchema = z.object({
  ...base,
  username: usernameSchema,
  password: passwordSchema,
});

export const adminUpdateSchema = z.object(base);

export type AdminCreateValues = z.infer<typeof adminCreateSchema>;
export type AdminUpdateValues = z.infer<typeof adminUpdateSchema>;
