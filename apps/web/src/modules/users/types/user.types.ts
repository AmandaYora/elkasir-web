export type AdminRole = "owner" | "admin" | "manager" | "viewer";
export type ActiveStatus = "active" | "inactive";

export interface AdminUser {
  id: string;
  name: string;
  email: string;
  username?: string;
  role: AdminRole;
  status: ActiveStatus;
  lastActiveAt?: string;
  createdAt: string;
}

export interface AdminCreateInput {
  name: string;
  email: string;
  username: string;
  password: string;
  role: AdminRole;
  status: ActiveStatus;
}

export interface AdminUpdateInput {
  name: string;
  email: string;
  role: AdminRole;
  status: ActiveStatus;
}
