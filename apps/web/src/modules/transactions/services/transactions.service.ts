import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { Transaction } from "@/modules/transactions/types/transaction.types";
import type { ListQuery } from "@/shared/types/pagination";

// Read-only service: transactions are a history view (list + detail), no mutations.
export const transactionsService = {
  list: (query?: ListQuery) => api.getPage<Transaction>(endpoints.transactions, { query }),
  get: (id: string) => api.get<Transaction>(`${endpoints.transactions}/${id}`),
};
