export type TransactionSource = "cashier" | "self_order";
export type PaymentMethod = "cash" | "qris";

export interface TransactionItem {
  productId?: string;
  productName: string;
  category: string;
  price: number;
  quantity: number;
  lineTotal: number;
  note?: string;
}

export interface Transaction {
  id: string;
  code: string;
  shiftId?: string;
  tableId?: string;
  selfOrderId?: string;
  cashierId?: string;
  orderType: string;
  source: TransactionSource;
  paymentMethod: PaymentMethod;
  status: string;
  subtotal: number;
  discount: number;
  tax: number;
  total: number;
  amountReceived: number;
  changeAmount: number;
  customerNote?: string;
  voidedAt?: string;
  voidReason?: string;
  createdAt: string;
  items: TransactionItem[];
}
