export type TableStatus = "active" | "inactive";

export interface DiningTable {
  id: string;
  code: string;
  name: string;
  area: string;
  seats: number;
  status: TableStatus;
  createdAt: string;
}

export interface TableInput {
  code: string;
  name: string;
  area: string;
  seats: number;
  status: TableStatus;
}
