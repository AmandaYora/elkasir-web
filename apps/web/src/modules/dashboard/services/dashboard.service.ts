import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { ListQuery } from "@/shared/types/pagination";
import type {
  CategorySales,
  DashboardReport,
  PaymentDistribution,
  SalesDay,
  TopProduct,
} from "@/modules/dashboard/types/dashboard.types";

// Report endpoints return plain objects/arrays (NOT paginated), so use api.get<T>.
export const dashboardService = {
  dashboard: () => api.get<DashboardReport>(endpoints.reports.dashboard),
  sales: (query?: ListQuery) => api.get<SalesDay[]>(endpoints.reports.sales, { query }),
  paymentDistribution: () => api.get<PaymentDistribution>(endpoints.reports.paymentDistribution),
  topProducts: (query?: ListQuery) =>
    api.get<TopProduct[]>(endpoints.reports.topProducts, { query }),
  salesByCategory: (query?: ListQuery) =>
    api.get<CategorySales[]>(endpoints.reports.salesByCategory, { query }),
};
