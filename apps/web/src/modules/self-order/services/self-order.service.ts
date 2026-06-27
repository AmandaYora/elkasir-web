import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { SelfOrder, SelfOrderStatus } from "@/modules/self-order/types/self-order.types";
import type { ListQuery } from "@/shared/types/pagination";

// Admin (authenticated) self-order service for the "Pesanan Masuk" monitor screen.
// Redeem + cash checkout is a POS till operation (staff-only) — not exposed to the web admin.
export const selfOrderService = {
  list: (query?: ListQuery) => api.getPage<SelfOrder>(endpoints.selfOrders, { query }),
  updateStatus: (id: string, status: SelfOrderStatus) =>
    api.patch<SelfOrder>(`${endpoints.selfOrders}/${id}/status`, { status }),
};
