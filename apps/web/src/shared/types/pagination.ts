// Standard pagination metadata (matches the backend `meta` envelope).
export interface PaginationMeta {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

// A page of results returned by list endpoints.
export interface Page<T> {
  data: T[];
  meta: PaginationMeta;
}

// Query params accepted by list endpoints.
export type ListQuery = Record<string, string | number | boolean | undefined | null>;
