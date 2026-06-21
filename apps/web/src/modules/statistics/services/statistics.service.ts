import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { ListQuery } from "@/shared/types/pagination";
import type {
  CategorySales,
  PaymentDistribution,
  SalesDay,
  StaffPerformance,
  TopProduct,
} from "@/modules/statistics/types/statistics.types";

// Report endpoints return plain objects/arrays (NOT paginated), so use api.get<T>.
export const statisticsService = {
  sales: (query?: ListQuery) => api.get<SalesDay[]>(endpoints.reports.sales, { query }),
  topProducts: (query?: ListQuery) => api.get<TopProduct[]>(endpoints.reports.topProducts, { query }),
  salesByCategory: (query?: ListQuery) =>
    api.get<CategorySales[]>(endpoints.reports.salesByCategory, { query }),
  paymentDistribution: () => api.get<PaymentDistribution>(endpoints.reports.paymentDistribution),
  staffPerformance: (query?: ListQuery) =>
    api.get<StaffPerformance[]>(endpoints.reports.staffPerformance, { query }),
};
