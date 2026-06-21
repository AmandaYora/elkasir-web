// @elkasir/shared — truly reusable, domain-agnostic cross-app code only.
// Keep this free of business rules (no order/product/payment logic). Put generic
// helpers, constants, or wire-format types shared across TS apps here.

/** Standard API success envelope used across Elkasir services. */
export interface ApiSuccess<T> {
  success: true;
  message: string;
  data: T;
}

/** Standard paginated envelope. */
export interface ApiPaginated<T> {
  success: true;
  message: string;
  data: T[];
  meta: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

/** Standard API error envelope. */
export interface ApiFailure {
  success: false;
  message: string;
  errors?: unknown[];
}
