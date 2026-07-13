import { z } from "zod";

export const createTenantSchema = z.object({
  storeName: z.string().min(1, "Nama toko wajib diisi"),
  storeSlug: z
    .string()
    .min(1, "Slug wajib diisi")
    .regex(/^[a-z0-9]+(-[a-z0-9]+)*$/, "Slug hanya boleh huruf kecil, angka, dan tanda hubung"),
  ownerName: z.string().min(1, "Nama pemilik wajib diisi"),
  ownerEmail: z.string().min(1, "Email pemilik wajib diisi").email("Email tidak valid"),
  ownerPassword: z.string().min(6, "Password minimal 6 karakter"),
});

export type CreateTenantFormValues = z.infer<typeof createTenantSchema>;
