export type StaffRole = "cashier" | "supervisor";
export type ActiveStatus = "active" | "inactive";

export interface Staff {
  id: string;
  name: string;
  username: string;
  email?: string;
  role: StaffRole;
  status: ActiveStatus;
  createdAt: string;
}

export interface StaffCreateInput {
  name: string;
  username: string;
  email?: string;
  password: string;
  role: StaffRole;
  status: ActiveStatus;
}

export interface StaffUpdateInput {
  name: string;
  username: string;
  email?: string;
  role: StaffRole;
  status: ActiveStatus;
}
