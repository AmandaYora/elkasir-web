import { z } from "zod";

export const cashMovementSchema = z
  .object({
    type: z.enum(["capital", "expense", "adjustment"]),
    amount: z.number().refine((n) => !Number.isNaN(n), "Nominal tidak valid."),
    notes: z.string().optional(),
    approvedBy: z.string().optional(),
  })
  .superRefine((val, ctx) => {
    if ((val.type === "capital" || val.type === "expense") && val.amount <= 0) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Nominal harus lebih dari 0.",
        path: ["amount"],
      });
    }
    if (val.type === "adjustment" && val.amount === 0) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Nominal penyesuaian tidak boleh 0.",
        path: ["amount"],
      });
    }
  });

export type CashMovementFormValues = z.infer<typeof cashMovementSchema>;
