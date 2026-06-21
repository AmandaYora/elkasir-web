import type { PaginationMeta } from "./pagination";

// Standard API envelopes (matches apps/api internal/platform/httpx).
export interface ApiSuccess<T> {
  success: true;
  message: string;
  data: T;
}

export interface ApiPaginated<T> {
  success: true;
  message: string;
  data: T[];
  meta: PaginationMeta;
}

export interface ApiErrorItem {
  code?: string;
  details?: unknown;
}

export interface ApiFailure {
  success: false;
  message: string;
  errors?: ApiErrorItem[];
}

// Normalized error thrown by the http-client for any failed request.
export class ApiError extends Error {
  status: number;
  code: string;
  errors?: ApiErrorItem[];

  constructor(status: number, message: string, code = "internal", errors?: ApiErrorItem[]) {
    super(message || "Terjadi kesalahan.");
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.errors = errors;
  }
}
