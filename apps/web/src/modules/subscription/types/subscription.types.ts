// Mirrors subscription/application.SubscriptionDTO / InvoiceDTO / CheckoutResult (camelCase).
export type SubscriptionStatus = "none" | "trial" | "active" | "past_due" | "expired" | "canceled";

export interface Subscription {
  status: SubscriptionStatus;
  planId?: string;
  // Resolved server-side regardless of the plan's active flag — may belong to a hidden/legacy
  // plan (e.g. "Premium Contributor") that no longer appears in `listPlans()`'s active-only
  // result, so these can't be looked up from the plans list — use them directly.
  planName?: string;
  planPrice?: number;
  planPeriodDays?: number;
  // Drives hiding "other plans"/upgrade options explicitly — see SubscriptionPage's use.
  planRenewalOnly?: boolean;
  currentPeriodStart?: string;
  currentPeriodEnd?: string;
}

export interface Plan {
  id: string;
  code: string;
  name: string;
  price: number;
  periodDays: number;
  isActive: boolean;
}

export type InvoiceStatus = "pending" | "paid" | "expired" | "failed";

export interface SubscriptionInvoice {
  id: string;
  planId: string;
  amount: number;
  status: InvoiceStatus;
  periodStart?: string;
  periodEnd?: string;
  createdAt: string;
}

export interface CheckoutResult {
  invoice: SubscriptionInvoice;
  qrString?: string;
  qrImageUrl?: string;
  simulated?: boolean;
}
