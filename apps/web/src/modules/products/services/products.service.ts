import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { CategoryOption, Product, ProductInput } from "@/modules/products/types/product.types";
import type { ListQuery } from "@/shared/types/pagination";

export const productsService = {
  list: (query?: ListQuery) => api.getPage<Product>(endpoints.products, { query }),
  get: (id: string) => api.get<Product>(`${endpoints.products}/${id}`),
  create: (body: ProductInput) => api.post<Product>(endpoints.products, body),
  update: (id: string, body: ProductInput) => api.put<Product>(`${endpoints.products}/${id}`, body),
  remove: (id: string) => api.del(`${endpoints.products}/${id}`),
  adjustStock: (id: string, delta: number) =>
    api.post<Product>(`${endpoints.products}/${id}/adjust-stock`, { delta }),
  // Category options for the product form (read-only helper; products owns its own fetch).
  listCategories: () =>
    api.getPage<CategoryOption>(endpoints.categories, { query: { limit: 200 } }),
};
