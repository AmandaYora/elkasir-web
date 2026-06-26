import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { Transaction } from "@/modules/transactions/types/transaction.types";
import type { ListQuery } from "@/shared/types/pagination";

// History view (list + detail) plus void: cancel a cash transaction on the running shift
// (restocks items + reverses its shift/report contribution server-side).
export const transactionsService = {
  list: (query?: ListQuery) => api.getPage<Transaction>(endpoints.transactions, { query }),
  get: (id: string) => api.get<Transaction>(`${endpoints.transactions}/${id}`),
  void: (id: string, reason: string) =>
    api.post<Transaction>(`${endpoints.transactions}/${id}/void`, { reason }),
};
