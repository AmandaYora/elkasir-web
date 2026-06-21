import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type {
  AdminUser,
  AdminCreateInput,
  AdminUpdateInput,
} from "@/modules/users/types/user.types";
import type { ListQuery } from "@/shared/types/pagination";

export const usersService = {
  list: (query?: ListQuery) => api.getPage<AdminUser>(endpoints.adminUsers, { query }),
  create: (body: AdminCreateInput) => api.post<AdminUser>(endpoints.adminUsers, body),
  update: (id: string, body: AdminUpdateInput) =>
    api.put<AdminUser>(`${endpoints.adminUsers}/${id}`, body),
  resetPassword: (id: string, password: string) =>
    api.post<void>(`${endpoints.adminUsers}/${id}/reset-password`, { password }),
  remove: (id: string) => api.del(`${endpoints.adminUsers}/${id}`),
};
