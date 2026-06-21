import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type {
  CheckoutResult,
  SelfOrder,
  SelfOrderStatus,
} from "@/modules/self-order/types/self-order.types";
import type { ListQuery } from "@/shared/types/pagination";

// Admin (authenticated) self-order service. Used by the "Pesanan Masuk" screen.
export const selfOrderService = {
  list: (query?: ListQuery) => api.getPage<SelfOrder>(endpoints.selfOrders, { query }),
  updateStatus: (id: string, status: SelfOrderStatus) =>
    api.patch<SelfOrder>(`${endpoints.selfOrders}/${id}/status`, { status }),
  redeem: (claimCode: string) =>
    api.get<SelfOrder>(`${endpoints.selfOrders}/redeem/${encodeURIComponent(claimCode)}`),
  // Checkout is idempotent via order state (paymentStatus=paid → replay); no Idempotency-Key needed.
  redeemCheckout: (claimCode: string) =>
    api.post<CheckoutResult>(
      `${endpoints.selfOrders}/redeem/${encodeURIComponent(claimCode)}/checkout`,
      undefined,
    ),
};
