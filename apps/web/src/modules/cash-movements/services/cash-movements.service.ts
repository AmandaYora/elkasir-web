import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { CashMovement } from "@/modules/cash-movements/types/cash-movement.types";
import type { ListQuery } from "@/shared/types/pagination";

// Read-only in the web admin: cash movements are RECORDED at the POS (supervisor); here we monitor.
export const cashMovementsService = {
  list: (query?: ListQuery) => api.getPage<CashMovement>(endpoints.cashMovements, { query }),
};
