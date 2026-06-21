import { z } from "zod";

export const withdrawalSchema = z.object({
  amount: z.number().int().positive("Jumlah penarikan harus lebih dari 0."),
  bank: z.string().min(1, "Bank wajib diisi."),
  account: z.string().min(1, "Nomor rekening wajib diisi."),
  holder: z.string().min(1, "Pemilik rekening wajib diisi."),
});

export type WithdrawalFormValues = z.infer<typeof withdrawalSchema>;
