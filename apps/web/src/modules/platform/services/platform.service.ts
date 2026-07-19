import { api, type RequestConfig } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type { ListQuery } from "@/shared/types/pagination";
import type {
  Tenant,
  CreateTenantInput,
  Plan,
  PlanInput,
  RevenueSummary,
  TenantRevenue,
  WithdrawalView,
  PlatformUser,
  CreatePlatformUserInput,
  GatewayConfig,
  UpdateGatewayConfigInput,
} from "@/modules/platform/types/platform.types";

// Every call passes tokenDomain:"platform" — this module never touches the tenant session.
const cfg: RequestConfig = { tokenDomain: "platform" };

export const platformService = {
  listTenants: () => api.get<Tenant[]>(endpoints.platform.tenants, cfg),
  createTenant: (body: CreateTenantInput) =>
    api.post<Tenant>(endpoints.platform.tenants, body, cfg),
  setTenantStatus: (id: string, status: "active" | "suspended") =>
    api.patch<Tenant>(`${endpoints.platform.tenants}/${id}/status`, { status }, cfg),
  tenantsRevenue: () => api.get<TenantRevenue[]>(endpoints.platform.tenantsRevenue, cfg),

  revenue: () => api.get<RevenueSummary>(endpoints.platform.revenue, cfg),

  listPlans: () => api.get<Plan[]>(endpoints.platform.plans, cfg),
  createPlan: (body: PlanInput) => api.post<Plan>(endpoints.platform.plans, body, cfg),
  updatePlan: (id: string, body: PlanInput) =>
    api.patch<Plan>(`${endpoints.platform.plans}/${id}`, body, cfg),

  listActiveWithdrawals: () => api.get<WithdrawalView[]>(endpoints.platform.withdrawals, cfg),
  claimWithdrawal: (id: string) =>
    api.patch<void>(`${endpoints.platform.withdrawals}/${id}/claim`, undefined, cfg),
  completeWithdrawal: (id: string) =>
    api.patch<void>(`${endpoints.platform.withdrawals}/${id}/success`, undefined, cfg),
  rejectWithdrawal: (id: string, reason: string) =>
    api.patch<void>(`${endpoints.platform.withdrawals}/${id}/reject`, { reason }, cfg),
  withdrawalHistory: (query?: ListQuery) =>
    api.getPage<WithdrawalView>(endpoints.platform.withdrawalHistory, { ...cfg, query }),

  listUsers: () => api.get<PlatformUser[]>(endpoints.platform.users, cfg),
  createUser: (body: CreatePlatformUserInput) =>
    api.post<PlatformUser>(endpoints.platform.users, body, cfg),
  setUserStatus: (id: string, status: "active" | "inactive") =>
    api.patch<void>(`${endpoints.platform.users}/${id}/status`, { status }, cfg),
  resetUserPassword: (id: string, password: string) =>
    api.patch<void>(`${endpoints.platform.users}/${id}/reset-password`, { password }, cfg),

  getPaymentConfig: () => api.get<GatewayConfig>(endpoints.platform.paymentConfig, cfg),
  updatePaymentConfig: (body: UpdateGatewayConfigInput) =>
    api.put<GatewayConfig>(endpoints.platform.paymentConfig, body, cfg),
};
