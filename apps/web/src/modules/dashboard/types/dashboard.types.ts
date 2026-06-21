// Report payload types for the dashboard module (camelCase, mirrors backend DTOs).

export interface DashboardSummary {
  txCount: number;
  revenue: number;
  salesTotal: number; // penjualan (subtotal − diskon)
  serviceTotal: number; // layanan (service + biaya gateway)
  taxTotal: number; // pajak (PPN)
  cashTotal: number;
  qrisTotal: number;
}

export interface DashboardRecentTransaction {
  id: string;
  code: string;
  source: string;
  paymentMethod: string;
  total: number;
  createdAt: string;
}

export interface DashboardReport {
  summary: DashboardSummary;
  recent: DashboardRecentTransaction[];
}

export interface SalesDay {
  day: string;
  txCount: number;
  revenue: number;
}

export interface TopProduct {
  productName: string;
  qty: number;
  revenue: number;
}

export interface CategorySales {
  category: string;
  revenue: number;
  qty: number;
}

export interface PaymentDistribution {
  cashTotal: number;
  qrisTotal: number;
  cashCount: number;
  qrisCount: number;
}
