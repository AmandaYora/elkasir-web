// Generic, domain-agnostic shared types.
export type ID = string;
export type ISODateString = string;

// Generic active/inactive status used by several resources.
export type ActiveStatus = "active" | "inactive";

export interface Option<T = string> {
  label: string;
  value: T;
}
