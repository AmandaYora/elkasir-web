import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type {
  PlaceOrderInput,
  PlaceResult,
  PublicMenu,
  PublicSelfOrderStatus,
  QuoteResult,
} from "@/modules/self-order/types/self-order.types";

// Public (no-auth) order service for the customer self-order page.
// Every call passes { auth: false } so no Bearer token is attached.
export const publicOrderService = {
  menu: (tableCode: string) =>
    api.get<PublicMenu>(`${endpoints.publicOrder}/${encodeURIComponent(tableCode)}`, {
      auth: false,
    }),
  place: (tableCode: string, body: PlaceOrderInput) =>
    api.post<PlaceResult>(`${endpoints.publicOrder}/${encodeURIComponent(tableCode)}`, body, {
      auth: false,
    }),
  quote: (tableCode: string, body: PlaceOrderInput) =>
    api.post<QuoteResult>(`${endpoints.publicOrder}/${encodeURIComponent(tableCode)}/quote`, body, {
      auth: false,
    }),
  status: (selfOrderId: string) =>
    api.get<PublicSelfOrderStatus>(`${endpoints.publicOrder}/status/${selfOrderId}`, {
      auth: false,
    }),
  simulatePaid: (selfOrderId: string) =>
    api.post<void>(`${endpoints.publicOrder}/${selfOrderId}/simulate-paid`, undefined, {
      auth: false,
    }),
};
