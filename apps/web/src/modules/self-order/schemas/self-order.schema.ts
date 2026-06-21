import { z } from "zod";

// Validation for the customer "place order" payload.
export const placeOrderItemSchema = z.object({
  productId: z.string().min(1, "Produk tidak valid"),
  quantity: z.number().int().min(1, "Jumlah minimal 1"),
  note: z.string().optional(),
});

export const placeOrderSchema = z.object({
  items: z.array(placeOrderItemSchema).min(1, "Pilih minimal satu menu"),
  paymentMethod: z.enum(["qris", "cash"]),
  customerNote: z.string().optional(),
});

export type PlaceOrderValues = z.infer<typeof placeOrderSchema>;

// Validation for the staff "redeem claim code" form.
export const redeemSchema = z.object({
  claimCode: z.string().min(1, "Kode klaim wajib diisi"),
});

export type RedeemValues = z.infer<typeof redeemSchema>;
