export interface Category {
  id: string;
  name: string;
  sortOrder: number;
  productCount: number;
  createdAt: string;
}

export interface CategoryInput {
  name: string;
  sortOrder?: number;
}
