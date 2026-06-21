import { z } from "zod";

export const productSchema = z.object({
  name: z.string().min(1, "Nama produk wajib diisi"),
  sku: z.string().min(1, "SKU wajib diisi"),
  categoryId: z.string().optional(),
  price: z.number().min(0, "Harga tidak boleh negatif"),
  cost: z.number().min(0, "Modal tidak boleh negatif"),
  stock: z.number().int().min(0, "Stok tidak boleh negatif"),
  status: z.enum(["active", "inactive"]),
  imageUrl: z.string().url("URL gambar tidak valid").optional().or(z.literal("")),
});

export type ProductFormValues = z.infer<typeof productSchema>;
