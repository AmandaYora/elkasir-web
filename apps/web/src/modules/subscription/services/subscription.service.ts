import { api } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import type {
  Subscription,
  Plan,
  CheckoutResult,
} from "@/modules/subscription/types/subscription.types";

// Tenant domain by default — never passes tokenDomain explicitly (matches every existing
// tenant-facing service in this app).
export const subscriptionService = {
  getCurrent: () => api.get<Subscription>(endpoints.subscription.root),
  listPlans: () => api.get<Plan[]>(endpoints.subscription.plans),
  checkout: (planId: string) =>
    api.post<CheckoutResult>(endpoints.subscription.checkout, { planId }),
};
