export type CashMovementType = "capital" | "expense" | "adjustment";

export interface CashMovement {
  id: string;
  shiftId?: string;
  type: CashMovementType;
  amount: number;
  notes?: string;
  createdBy?: string;
  approvedBy?: string;
  createdAt: string;
}

export interface CashMovementInput {
  type: CashMovementType;
  amount: number;
  notes?: string;
  approvedBy?: string;
}
