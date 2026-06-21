import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { DiningTable, TableInput } from "@/modules/tables/types/table.types";
import type { ListQuery } from "@/shared/types/pagination";

export const tablesService = {
  list: (query?: ListQuery) => api.getPage<DiningTable>(endpoints.tables, { query }),
  get: (id: string) => api.get<DiningTable>(`${endpoints.tables}/${id}`),
  create: (body: TableInput) => api.post<DiningTable>(endpoints.tables, body),
  update: (id: string, body: TableInput) => api.put<DiningTable>(`${endpoints.tables}/${id}`, body),
  remove: (id: string) => api.del(`${endpoints.tables}/${id}`),
};
