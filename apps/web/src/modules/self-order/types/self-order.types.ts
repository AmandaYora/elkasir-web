// Self-order domain types (camelCase, aligned with backend DTOs).

export type SelfOrderStatus = "placed" | "preparing" | "completed";
export type SelfOrderPaymentMethod = "qris" | "cash";
export type SelfOrderPaymentStatus = "pending" | "paid" | "unpaid" | "expired" | "failed";

// Indonesian labels for the preparation stage (used in toasts and badges).
export const ORDER_STAGE_LABEL: Record<SelfOrderStatus, string> = {
  placed: "Masuk",
  preparing: "Disiapkan",
  completed: "Selesai",
};

export interface SelfOrderItem {
  productName: string;
  category: string;
  price: number;
  quantity: number;
  lineTotal: number;
  note?: string;
}

export interface SelfOrder {
  id: string;
  tableCode: string;
  tableName: string;
  status: SelfOrderStatus;
  paymentMethod: SelfOrderPaymentMethod;
  paymentStatus: SelfOrderPaymentStatus;
  claimCode?: string;
  subtotal: number;
  total: number;
  customerNote?: string;
  transactionId?: string;
  createdAt: string;
  items: SelfOrderItem[];
}

export interface CheckoutResult {
  transactionId: string;
  order: SelfOrder;
}

// ── Public (customer) order types ────────────────────────────
export interface PublicMenuProduct {
  id: string;
  name: string;
  category: string;
  price: number;
  imageUrl?: string;
}

export interface PublicMenu {
  table: { code: string; name: string; area: string; status: string };
  categories: string[];
  products: PublicMenuProduct[];
}

export interface PlaceOrderItem {
  productId: string;
  quantity: number;
  note?: string;
}

export interface PlaceOrderInput {
  items: PlaceOrderItem[];
  paymentMethod: SelfOrderPaymentMethod;
  customerNote?: string;
}

export interface PlaceResult {
  order: SelfOrder;
  qrString?: string;
  claimCode?: string;
  simulated?: boolean;
}

export interface PublicSelfOrderStatus {
  id: string;
  status: string;
  paymentStatus: string;
  total: number;
}
