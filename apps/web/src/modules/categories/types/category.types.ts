export interface Category {
  id: string;
  name: string;
  productCount: number;
  createdAt: string;
}

export interface CategoryInput {
  name: string;
}
