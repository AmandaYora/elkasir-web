import { z } from "zod";

// Validates the settings form before PATCH. Percent fields clamped 0–100.
export const settingsSchema = z
  .object({
    storeName: z.string().trim().min(1, "Nama toko wajib diisi.").max(150),
    storePhone: z.string().trim().max(40),
    storeAddress: z.string().trim().max(255),
    storeLogoUrl: z.string().trim().max(500),
    maxDiscountPercent: z.number().int().min(0).max(100),
    maxOperationalExpense: z.number().int().min(0),
    cashVarianceTolerance: z.number().int().min(0),
    featureSelfOrder: z.boolean(),
    featureQris: z.boolean(),
    featurePayAtCashier: z.boolean(),
    taxEnabled: z.boolean(),
    taxPercent: z.number().int().min(0).max(100),
    servicePercent: z.number().int().min(0).max(100),
  })
  // Selaras dengan guard backend: saat self-order aktif, minimal satu metode bayar harus aktif.
  .refine((s) => !s.featureSelfOrder || s.featureQris || s.featurePayAtCashier, {
    message: "Minimal satu metode pembayaran harus aktif saat self-order aktif.",
    path: ["featureQris"],
  });

export type SettingsValues = z.infer<typeof settingsSchema>;
