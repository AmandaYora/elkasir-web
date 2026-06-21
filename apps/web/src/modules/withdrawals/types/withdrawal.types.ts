export interface Withdrawal {
  id: string;
  amount: number;
  bank: string;
  account: string;
  holder: string;
  status: string;
  reference?: string;
  requestedBy?: string;
  createdAt: string;
}

export interface WithdrawalInput {
  amount: number;
  bank: string;
  account: string;
  holder: string;
}
