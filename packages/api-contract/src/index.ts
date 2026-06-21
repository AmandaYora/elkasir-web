import createClient, { type Client } from "openapi-fetch";
import type { paths } from "../generated/ts/schema";

export type ApiClient = Client<paths>;

/**
 * Factory client API typed dari kontrak OpenAPI.
 * - `baseUrl`   : alamat API (mis. import.meta.env.VITE_API_BASE_URL).
 * - `getToken`  : penyedia access token (disuntik sebagai Bearer per request).
 */
export function createApiClient(opts: {
  baseUrl: string;
  getToken?: () => string | null | undefined;
}): ApiClient {
  const client = createClient<paths>({ baseUrl: opts.baseUrl });

  if (opts.getToken) {
    client.use({
      onRequest({ request }) {
        const token = opts.getToken?.();
        if (token) request.headers.set("Authorization", `Bearer ${token}`);
        return request;
      },
    });
  }

  return client;
}

export type { paths, components } from "../generated/ts/schema";
