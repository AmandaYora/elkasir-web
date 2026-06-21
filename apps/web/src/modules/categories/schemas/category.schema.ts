import { z } from "zod";

export const categorySchema = z.object({
  name: z.string().min(1, "Nama kategori wajib diisi"),
  sortOrder: z.number().int().optional(),
});

export type CategoryFormValues = z.infer<typeof categorySchema>;
