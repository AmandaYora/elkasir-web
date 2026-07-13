import { z } from "zod";

export const planSchema = z.object({
  code: z.string().min(1, "Kode paket wajib diisi"),
  name: z.string().min(1, "Nama paket wajib diisi"),
  price: z.number().int("Harga harus bilangan bulat").positive("Harga harus lebih dari 0"),
  periodDays: z.number().int().positive("Periode (hari) harus lebih dari 0"),
  isActive: z.boolean(),
});

export type PlanFormValues = z.infer<typeof planSchema>;
