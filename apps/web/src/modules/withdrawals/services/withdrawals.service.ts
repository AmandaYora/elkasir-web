import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { Withdrawal, WithdrawalInput } from "@/modules/withdrawals/types/withdrawal.types";
import type { ListQuery } from "@/shared/types/pagination";

export const withdrawalsService = {
  list: (query?: ListQuery) => api.getPage<Withdrawal>(endpoints.withdrawals, { query }),
  create: (body: WithdrawalInput) => api.post<Withdrawal>(endpoints.withdrawals, body),
};
