import { z } from "zod";

// Validates the settings form before PATCH. Percent fields clamped 0–100.
export const settingsSchema = z.object({
  maxDiscountPercent: z.number().int().min(0).max(100),
  maxOperationalExpense: z.number().int().min(0),
  cashVarianceTolerance: z.number().int().min(0),
  featureSelfOrder: z.boolean(),
  featureQris: z.boolean(),
  taxEnabled: z.boolean(),
  taxPercent: z.number().int().min(0).max(100),
  servicePercent: z.number().int().min(0).max(100),
});

export type SettingsValues = z.infer<typeof settingsSchema>;
