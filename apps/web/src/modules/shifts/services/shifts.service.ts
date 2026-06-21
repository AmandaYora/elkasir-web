import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { Shift } from "@/modules/shifts/types/shift.types";
import type { ListQuery } from "@/shared/types/pagination";

// Read-only service: shifts are a history / reconciliation view (list + detail).
export const shiftsService = {
  list: (query?: ListQuery) => api.getPage<Shift>(endpoints.shifts, { query }),
  get: (id: string) => api.get<Shift>(`${endpoints.shifts}/${id}`),
};
