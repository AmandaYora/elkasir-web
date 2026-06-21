import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { Category, CategoryInput } from "@/modules/categories/types/category.types";
import type { ListQuery } from "@/shared/types/pagination";

export const categoriesService = {
  list: (query?: ListQuery) => api.getPage<Category>(endpoints.categories, { query }),
  create: (body: CategoryInput) => api.post<Category>(endpoints.categories, body),
  update: (id: string, body: CategoryInput) => api.put<Category>(`${endpoints.categories}/${id}`, body),
  remove: (id: string) => api.del(`${endpoints.categories}/${id}`),
};
