import { z } from "zod";

export const tableSchema = z.object({
  code: z.string().min(1, "Kode meja wajib diisi"),
  name: z.string().min(1, "Nama meja wajib diisi"),
  area: z.string().min(1, "Area wajib diisi"),
  seats: z.number().int().min(0, "Kursi tidak boleh negatif"),
  status: z.enum(["active", "inactive"]),
});

export type TableFormValues = z.infer<typeof tableSchema>;
