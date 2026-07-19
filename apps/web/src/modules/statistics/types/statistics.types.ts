// Report payload types used by the statistics module (camelCase, mirrors backend DTOs).

export interface SalesDay {
  day: string;
  txCount: number;
  revenue: number;
}

export interface SalesMonth {
  month: string;
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

export interface StaffPerformance {
  staffId: string;
  name: string;
  txCount: number;
  revenue: number;
}
