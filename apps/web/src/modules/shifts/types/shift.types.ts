export type ShiftStatus = "open" | "closed";

export interface Shift {
  id: string;
  staffId: string;
  status: ShiftStatus;
  initialCash: number;
  cashSales: number;
  qrisSales: number;
  additionalCapital: number;
  expenses: number;
  withdrawals: number;
  adjustments: number;
  drawerOpenCount: number;
  expectedCash?: number;
  actualCash?: number;
  variance?: number;
  closeApprovedBy?: string;
  openedAt: string;
  closedAt?: string;
  createdAt: string;
}
