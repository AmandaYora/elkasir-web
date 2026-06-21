import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { Staff, StaffCreateInput, StaffUpdateInput } from "@/modules/staff/types/staff.types";
import type { ListQuery } from "@/shared/types/pagination";

export const staffService = {
  list: (query?: ListQuery) => api.getPage<Staff>(endpoints.staff, { query }),
  create: (body: StaffCreateInput) => api.post<Staff>(endpoints.staff, body),
  update: (id: string, body: StaffUpdateInput) => api.put<Staff>(`${endpoints.staff}/${id}`, body),
  resetPassword: (id: string, password: string) =>
    api.post<void>(`${endpoints.staff}/${id}/reset-password`, { password }),
  remove: (id: string) => api.del(`${endpoints.staff}/${id}`),
};
