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
  service: number; // biaya layanan 2% (rounded)
  gatewayFee: number; // biaya gateway QRIS (0 utk cash)
  serviceLine: number; // "Layanan" = service + gatewayFee
  tax: number; // PPN
  total: number;
  customerNote?: string;
  transactionId?: string;
  createdAt: string;
  items: SelfOrderItem[];
}

// Rincian biaya dari endpoint quote (ditampilkan sebelum pesanan dibuat).
export interface QuoteResult {
  subtotal: number;
  service: number;
  gatewayFee: number;
  serviceLine: number;
  tax: number;
  total: number;
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
  // Flag fitur toko: kontrol tampil/tidaknya halaman & tiap metode pembayaran.
  featureSelfOrder: boolean;
  featureQris: boolean;
  featurePayAtCashier: boolean;
  // Persen layanan & PPN agar rincian biaya bisa menjelaskan dirinya (mis. "Layanan (2%)").
  servicePercent: number;
  taxPercent: number;
  taxEnabled: boolean;
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
  qrImageUrl?: string;
  claimCode?: string;
  simulated?: boolean;
}

export interface PublicSelfOrderStatus {
  id: string;
  status: string;
  paymentStatus: string;
  total: number;
}
