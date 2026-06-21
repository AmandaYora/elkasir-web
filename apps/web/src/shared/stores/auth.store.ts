// Global session store (Zustand) — replaces the old React Context auth provider.
// Holds the authenticated user and drives login/logout/session restore. The http-client
// owns token storage; this store owns the user + session lifecycle.
import { create } from "zustand";
import { api, setOnUnauthorized, tokenStore } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";

export interface SessionUser {
  id: string;
  name: string;
  email: string;
  role: string;
  storeId: string;
  actor: string;
}

interface AuthUser {
  id: string;
  name: string;
  email?: string;
  role: string;
  storeId: string;
  actor: string;
}

interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: AuthUser;
}

type AuthStatus = "loading" | "ready";
type LoginResult = { ok: true } | { ok: false; error: string };

interface AuthState {
  user: SessionUser | null;
  status: AuthStatus;
  restore: () => Promise<void>;
  login: (email: string, password: string) => Promise<LoginResult>;
  logout: () => Promise<void>;
}

const toSession = (u: AuthUser): SessionUser => ({
  id: u.id,
  name: u.name,
  email: u.email ?? "",
  role: u.role,
  storeId: u.storeId,
  actor: u.actor,
});

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  status: "loading",

  restore: async () => {
    // Clear the session on an unrecoverable 401 (refresh failed).
    setOnUnauthorized(() => set({ user: null }));
    if (!tokenStore.access()) {
      set({ status: "ready" });
      return;
    }
    try {
      const me = await api.get<AuthUser>(endpoints.auth.me);
      set({ user: toSession(me), status: "ready" });
    } catch {
      tokenStore.clear();
      set({ user: null, status: "ready" });
    }
  },

  login: async (email, password) => {
    try {
      const res = await api.post<LoginResponse>(
        endpoints.auth.adminLogin,
        { email: email.trim(), password },
        { auth: false },
      );
      tokenStore.set(res.accessToken, res.refreshToken);
      set({ user: toSession(res.user) });
      return { ok: true };
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Tidak dapat masuk. Periksa koneksi lalu coba lagi.";
      return { ok: false, error: message };
    }
  },

  logout: async () => {
    const refreshToken = tokenStore.refresh();
    if (refreshToken) {
      try {
        await api.post(endpoints.auth.logout, { refreshToken }, { auth: false });
      } catch {
        /* best-effort */
      }
    }
    tokenStore.clear();
    set({ user: null });
  },
}));
