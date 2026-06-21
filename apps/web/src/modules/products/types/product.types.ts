export type ProductStatus = "active" | "inactive";

export interface Product {
  id: string;
  name: string;
  sku: string;
  categoryId?: string;
  category: string;
  price: number;
  cost: number;
  stock: number;
  status: ProductStatus;
  imageUrl?: string;
  createdAt: string;
}

export interface ProductInput {
  categoryId?: string;
  sku: string;
  name: string;
  price: number;
  cost: number;
  stock: number;
  status: ProductStatus;
  imageUrl?: string;
}

export interface CategoryOption {
  id: string;
  name: string;
}
