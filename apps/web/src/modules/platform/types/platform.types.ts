// Mirrors of every superadmin (Konsol Platform) DTO from PLAN.md §3 (camelCase, matching the Go
// json tags). Money amounts are plain `number` — same convention already used for tenant-facing
// amounts elsewhere in this app (backend BIGINT rupiah, well under Number.MAX_SAFE_INTEGER).

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  status: "active" | "suspended";
  createdAt: string;
}

export interface CreateTenantInput {
  storeName: string;
  storeSlug: string;
  ownerName: string;
  ownerEmail: string;
  ownerPassword: string;
}

// Cross-tenant admin-password-reset recovery flow — the superadmin's only way to restore access
// to a tenant that lost its own admin credentials (adminuser's own reset is self-service only).
export interface TenantAdmin {
  id: string;
  name: string;
  email: string;
  role: string;
}

export interface Plan {
  id: string;
  code: string;
  name: string;
  price: number;
  periodDays: number;
  isActive: boolean;
  // Read-only — set once at creation (e.g. the "Premium Contributor" legacy-backfill plan),
  // never part of PlanInput, so it can't be toggled via the create/edit form (mirrors `code`'s
  // immutability). A renewalOnly plan can only ever be renewed by a subscriber already on it.
  renewalOnly: boolean;
}

export interface PlanInput {
  code: string;
  name: string;
  price: number;
  periodDays: number;
  isActive: boolean;
}

export interface RevenueSummary {
  subscriptionRevenue: number;
  tenantAvailableBalance: number;
  totalMonitored: number;
}

export interface TenantRevenue {
  storeId: string;
  name: string;
  slug: string;
  balance: number;
}

export type WithdrawalStatus = "pending" | "processing" | "success" | "failed";

export interface WithdrawalView {
  id: string;
  storeId: string;
  amount: number;
  bank: string;
  account: string;
  holder: string;
  status: WithdrawalStatus;
  requestedBy?: string;
  processedBy?: string;
  claimedAt?: string;
  processedAt?: string;
  rejectedReason?: string;
  createdAt: string;
  tenantName: string;
  claimantName?: string;
}

export type PlatformUserStatus = "active" | "inactive";

export interface PlatformUser {
  id: string;
  name: string;
  email: string;
  status: PlatformUserStatus;
  createdAt: string;
}

export interface CreatePlatformUserInput {
  name: string;
  email: string;
  password: string;
}

// PLAN.md §9 (Part 2) — payment gateway config (Tripay/Midtrans, one wallet, DB-backed), plus
// (§11) ElProof — a SEPARATE, always-on wallet used only for subscription billing, not part of
// the provider switch above. The old app registry ("Aplikasi Terdaftar") was removed — Elkasir
// is no longer a payment-gateway-as-a-service provider for other apps (Part 3 deprecated).

export interface GatewayConfig {
  provider: "tripay" | "midtrans" | "";
  sandbox: boolean;
  tripayApiKeyMasked: string;
  tripayPrivateKeyMasked: string;
  tripayMerchantCode: string;
  tripayMethod: string;
  midtransServerKeyMasked: string;
  elproofAppId: string;
  elproofSecretMasked: string;
  elproofBaseUrl: string;
}

// Secret fields are OMITTED (not sent as "") when the superadmin doesn't retype them — the
// backend treats a present-but-empty field as "clear this secret" and an absent field as
// "keep the existing encrypted value" (§9.1.2). Build the request body accordingly.
export interface UpdateGatewayConfigInput {
  provider: "tripay" | "midtrans" | "";
  sandbox: boolean;
  tripayMethod: string;
  tripayApiKey?: string;
  tripayPrivateKey?: string;
  tripayMerchantCode?: string;
  midtransServerKey?: string;
  elproofAppId?: string;
  elproofSecret?: string;
  elproofBaseUrl: string;
}
