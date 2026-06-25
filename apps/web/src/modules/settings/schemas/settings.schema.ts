import { z } from "zod";

// Validates the settings form before PATCH. Percent fields clamped 0–100.
export const settingsSchema = z
  .object({
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
