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

// Exported so non-Axios transports (e.g. SSE/EventSource, which can't use this Axios
// instance) build URLs from the same base. Modules must not hardcode the base URL.
export const BASE_URL = (
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/api/v1"
).replace(/\/$/, "");

// Two fully separate identity domains (PLAN.md §2.1) — tenant (admin/staff) vs platform
// (superadmin, Konsol Platform). Every function below defaults to "tenant" so every existing
// call site (which predates this domain concept) keeps working unchanged.
export type TokenDomain = "tenant" | "platform";

const TOKEN_KEYS: Record<TokenDomain, { access: string; refresh: string }> = {
  tenant: { access: "elkasir_access_token", refresh: "elkasir_refresh_token" },
  platform: { access: "elkasir_platform_access_token", refresh: "elkasir_platform_refresh_token" },
};

export const tokenStore = {
  access: (domain: TokenDomain = "tenant") => storage.get(TOKEN_KEYS[domain].access),
  refresh: (domain: TokenDomain = "tenant") => storage.get(TOKEN_KEYS[domain].refresh),
  set(access: string, refresh: string, domain: TokenDomain = "tenant") {
    storage.set(TOKEN_KEYS[domain].access, access);
    storage.set(TOKEN_KEYS[domain].refresh, refresh);
  },
  clear(domain: TokenDomain = "tenant") {
    storage.remove(TOKEN_KEYS[domain].access);
    storage.remove(TOKEN_KEYS[domain].refresh);
  },
};

// Called when a domain's session cannot be recovered (refresh failed) — set by each domain's
// own auth store. Domain-keyed so a platform-session callback can't silently clobber the
// tenant session's (or vice versa) — see PLAN.md §1a.
const onUnauthorizedByDomain: Record<TokenDomain, (() => void) | null> = {
  tenant: null,
  platform: null,
};
export function setOnUnauthorized(cb: (() => void) | null, domain: TokenDomain = "tenant") {
  onUnauthorizedByDomain[domain] = cb;
}

// Called on a tenant-domain 402 (subscription package inactive, §2.15) — platform-domain
// requests are never gated by this, so there is no domain param here (unlike setOnUnauthorized).
let onPaymentRequired: (() => void) | null = null;
export function setOnPaymentRequired(cb: (() => void) | null) {
  onPaymentRequired = cb;
}

export const httpClient: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  timeout: 30000,
  headers: { "Content-Type": "application/json" },
});

type DomainConfig = InternalAxiosRequestConfig & {
  tokenDomain?: TokenDomain;
  _retried?: boolean;
};

// Attach the Bearer token for this request's domain (default "tenant"), unless the request
// opts out (auth: false via header marker).
httpClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const domain: TokenDomain = (config as DomainConfig).tokenDomain ?? "tenant";
  const skipAuth = config.headers?.["X-Skip-Auth"];
  if (skipAuth) {
    delete config.headers["X-Skip-Auth"];
  } else {
    const token = tokenStore.access(domain);
    if (token) config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Single in-flight refresh per domain, shared by concurrent 401s within that domain.
const refreshing: Record<TokenDomain, Promise<boolean> | null> = { tenant: null, platform: null };
async function refreshTokens(domain: TokenDomain): Promise<boolean> {
  if (refreshing[domain]) return refreshing[domain]!;
  const refreshToken = tokenStore.refresh(domain);
  if (!refreshToken) return false;
  refreshing[domain] = (async () => {
    try {
      const res = await axios.post(`${BASE_URL}/auth/refresh`, { refreshToken });
      const data = (res.data as ApiSuccess<{ accessToken: string; refreshToken: string }>).data;
      tokenStore.set(data.accessToken, data.refreshToken, domain);
      return true;
    } catch {
      return false;
    } finally {
      refreshing[domain] = null;
    }
  })();
  return refreshing[domain]!;
}

httpClient.interceptors.response.use(
  (res) => res,
  async (error: AxiosError<ApiFailure>) => {
    const original = error.config as DomainConfig | undefined;
    const domain: TokenDomain = original?.tokenDomain ?? "tenant";

    if (error.response?.status === 401 && original && !original._retried) {
      original._retried = true;
      if (await refreshTokens(domain)) {
        original.headers.Authorization = `Bearer ${tokenStore.access(domain)}`;
        return httpClient(original);
      }
      tokenStore.clear(domain);
      onUnauthorizedByDomain[domain]?.();
    }

    // New branch (§1a) — today only 401 has special handling above; there's no existing 403
    // branch this mirrors. Tenant-domain only: a locked-out package is a tenant-only concept.
    if (error.response?.status === 402 && domain === "tenant") {
      onPaymentRequired?.();
    }

    const body = error.response?.data;
    if (body && typeof body === "object" && "message" in body) {
      const code = body.errors?.[0]?.code ?? "internal";
      throw new ApiError(error.response?.status ?? 0, body.message, code, body.errors);
    }
    throw new ApiError(
      error.response?.status ?? 0,
      error.message || "Koneksi bermasalah. Periksa internet lalu coba lagi.",
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
// Bearer token for public requests) doesn't collide with it. Exported so module services can
// reference it explicitly (e.g. platform services always pass `{ tokenDomain: "platform" }`).
export type RequestConfig = Omit<AxiosRequestConfig, "auth"> & {
  query?: ListQuery;
  auth?: boolean;
  tokenDomain?: TokenDomain;
};

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
  // Multipart upload. Pass a FormData; axios/browser set the multipart boundary
  // automatically — but only if we clear the instance's default JSON Content-Type
  // first. Left as-is, Axios sees an explicit "application/json" already set and
  // JSON.stringifies the FormData instead of sending it as multipart, so the
  // server always fails to parse the form regardless of file size. Use
  // `onUploadProgress` (via config) for a progress bar.
  async upload<T>(url: string, form: FormData, config?: RequestConfig): Promise<T> {
    const cfg = withQuery(config);
    cfg.headers = { ...(cfg.headers ?? {}), "Content-Type": undefined };
    const res = await httpClient.post<ApiSuccess<T>>(url, form, cfg);
    return res.data?.data as T;
  },
};
