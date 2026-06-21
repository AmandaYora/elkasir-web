// Single Axios HTTP client for the whole app. Injects the Bearer token, refreshes on
// 401, and normalizes responses/errors to the standard envelope. Modules MUST use this
// client (via their services) — never create another Axios instance.
import axios, {
  AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from "axios";
import { storage } from "@/shared/lib/storage";
import { ApiError, type ApiFailure, type ApiPaginated, type ApiSuccess } from "@/shared/types/api";
import type { ListQuery, Page } from "@/shared/types/pagination";

const BASE_URL = ((import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/api/v1").replace(
  /\/$/,
  "",
);

const ACCESS_KEY = "elkasir_access_token";
const REFRESH_KEY = "elkasir_refresh_token";

export const tokenStore = {
  access: () => storage.get(ACCESS_KEY),
  refresh: () => storage.get(REFRESH_KEY),
  set(access: string, refresh: string) {
    storage.set(ACCESS_KEY, access);
    storage.set(REFRESH_KEY, refresh);
  },
  clear() {
    storage.remove(ACCESS_KEY);
    storage.remove(REFRESH_KEY);
  },
};

// Called when the session cannot be recovered (refresh failed) — set by the auth store.
let onUnauthorized: (() => void) | null = null;
export function setOnUnauthorized(cb: (() => void) | null) {
  onUnauthorized = cb;
}

export const httpClient: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  timeout: 30000,
  headers: { "Content-Type": "application/json" },
});

// Attach the Bearer token unless the request opts out (auth: false via header marker).
httpClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const skipAuth = config.headers?.["X-Skip-Auth"];
  if (skipAuth) {
    delete config.headers["X-Skip-Auth"];
  } else {
    const token = tokenStore.access();
    if (token) config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Single in-flight refresh shared by concurrent 401s.
let refreshing: Promise<boolean> | null = null;
async function refreshTokens(): Promise<boolean> {
  if (refreshing) return refreshing;
  const refreshToken = tokenStore.refresh();
  if (!refreshToken) return false;
  refreshing = (async () => {
    try {
      const res = await axios.post(`${BASE_URL}/auth/refresh`, { refreshToken });
      const data = (res.data as ApiSuccess<{ accessToken: string; refreshToken: string }>).data;
      tokenStore.set(data.accessToken, data.refreshToken);
      return true;
    } catch {
      return false;
    } finally {
      refreshing = null;
    }
  })();
  return refreshing;
}

httpClient.interceptors.response.use(
  (res) => res,
  async (error: AxiosError<ApiFailure>) => {
    const original = error.config as
      | (InternalAxiosRequestConfig & { _retried?: boolean })
      | undefined;

    if (error.response?.status === 401 && original && !original._retried) {
      original._retried = true;
      if (await refreshTokens()) {
        original.headers.Authorization = `Bearer ${tokenStore.access()}`;
        return httpClient(original);
      }
      tokenStore.clear();
      onUnauthorized?.();
    }

    const body = error.response?.data;
    if (body && typeof body === "object" && "message" in body) {
      const code = body.errors?.[0]?.code ?? "internal";
      throw new ApiError(error.response?.status ?? 0, body.message, code, body.errors);
    }
    throw new ApiError(
      error.response?.status ?? 0,
      error.message || "Tidak dapat terhubung ke server.",
    );
  },
);

function withQuery(config?: RequestConfig): AxiosRequestConfig {
  const { query, auth, ...rest } = config ?? {};
  const cfg: AxiosRequestConfig = { ...rest };
  if (query) {
    const params: Record<string, string> = {};
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null && v !== "") params[k] = String(v);
    }
    cfg.params = params;
  }
  if (auth === false) {
    cfg.headers = { ...(cfg.headers ?? {}), "X-Skip-Auth": "1" };
  }
  return cfg;
}

// Omit Axios's built-in `auth` (basic-credentials) so our boolean `auth` flag (skip the
// Bearer token for public requests) doesn't collide with it.
type RequestConfig = Omit<AxiosRequestConfig, "auth"> & { query?: ListQuery; auth?: boolean };

// Typed helpers that unwrap the standard envelope. `get`/`post`/... return the inner
// `data`; `getPage` returns `{ data, meta }` for list endpoints.
export const api = {
  async get<T>(url: string, config?: RequestConfig): Promise<T> {
    const res = await httpClient.get<ApiSuccess<T>>(url, withQuery(config));
    return res.data.data;
  },
  async getPage<T>(url: string, config?: RequestConfig): Promise<Page<T>> {
    const res = await httpClient.get<ApiPaginated<T>>(url, withQuery(config));
    return { data: res.data.data, meta: res.data.meta };
  },
  async post<T>(url: string, body?: unknown, config?: RequestConfig): Promise<T> {
    const res = await httpClient.post<ApiSuccess<T>>(url, body, withQuery(config));
    return res.data?.data as T;
  },
  async put<T>(url: string, body?: unknown, config?: RequestConfig): Promise<T> {
    const res = await httpClient.put<ApiSuccess<T>>(url, body, withQuery(config));
    return res.data?.data as T;
  },
  async patch<T>(url: string, body?: unknown, config?: RequestConfig): Promise<T> {
    const res = await httpClient.patch<ApiSuccess<T>>(url, body, withQuery(config));
    return res.data?.data as T;
  },
  async del<T = void>(url: string, config?: RequestConfig): Promise<T> {
    const res = await httpClient.delete<ApiSuccess<T>>(url, withQuery(config));
    return res.data?.data as T;
  },
};
