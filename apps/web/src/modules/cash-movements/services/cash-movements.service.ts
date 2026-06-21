import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { CashMovement, CashMovementInput } from "@/modules/cash-movements/types/cash-movement.types";
import type { ListQuery } from "@/shared/types/pagination";

export const cashMovementsService = {
  list: (query?: ListQuery) => api.getPage<CashMovement>(endpoints.cashMovements, { query }),
  create: (body: CashMovementInput) => api.post<CashMovement>(endpoints.cashMovements, body),
};
