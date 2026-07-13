// The 4 real statuses (§2.7 claim -> complete flow) — was a bare `string` (§1a); now a literal
// union so a drift like the old openapi.yaml enum (`[pending, paid, rejected]`, never actually
// returned by the backend) fails at compile time instead of silently rendering wrong.
export type WithdrawalStatus = "pending" | "processing" | "success" | "failed";

export interface Withdrawal {
  id: string;
  amount: number;
  bank: string;
  account: string;
  holder: string;
  status: WithdrawalStatus;
  reference?: string;
  requestedBy?: string;
  rejectedReason?: string;
  createdAt: string;
}

export interface WithdrawalInput {
  amount: number;
  bank: string;
  account: string;
  holder: string;
}
