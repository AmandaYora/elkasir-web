// Platform (superadmin) session store — twin of `useAuthStore`, fully isolated: separate
// Zustand store, separate token domain ("platform"), separate session-user shape (PLAN.md §2.1).
// Logging into one must never authenticate the other.
import { create } from "zustand";
import { api, setOnUnauthorized, tokenStore } from "@/shared/services/http-client";
import { endpoints } from "@/shared/services/api-endpoints";
import { ApiError } from "@/shared/types/api";

export interface PlatformSessionUser {
  id: string;
  name: string;
  email: string;
  actor: string;
}

interface PlatformAuthUser {
  id: string;
  name: string;
  email?: string;
  actor: string;
}

interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: PlatformAuthUser;
}

type AuthStatus = "loading" | "ready";
type LoginResult = { ok: true } | { ok: false; error: string };

interface PlatformAuthState {
  user: PlatformSessionUser | null;
  status: AuthStatus;
  restore: () => Promise<void>;
  login: (email: string, password: string) => Promise<LoginResult>;
  logout: () => Promise<void>;
}

const toSession = (u: PlatformAuthUser): PlatformSessionUser => ({
  id: u.id,
  name: u.name,
  email: u.email ?? "",
  actor: u.actor,
});

export const usePlatformAuthStore = create<PlatformAuthState>((set) => ({
  user: null,
  status: "loading",

  restore: async () => {
    // Clear the session on an unrecoverable 401 (refresh failed) — domain-keyed so this never
    // touches the tenant session's callback.
    setOnUnauthorized(() => set({ user: null }), "platform");
    if (!tokenStore.access("platform")) {
      set({ status: "ready" });
      return;
    }
    try {
      const me = await api.get<PlatformAuthUser>(endpoints.auth.me, { tokenDomain: "platform" });
      if (me.actor !== "platform") {
        // Defensive: a token in the platform slot that somehow doesn't carry a platform
        // principal is not a valid platform session.
        tokenStore.clear("platform");
        set({ user: null, status: "ready" });
        return;
      }
      set({ user: toSession(me), status: "ready" });
    } catch {
      tokenStore.clear("platform");
      set({ user: null, status: "ready" });
    }
  },

  login: async (email, password) => {
    try {
      const res = await api.post<LoginResponse>(
        endpoints.auth.platformLogin,
        { email: email.trim(), password },
        { auth: false, tokenDomain: "platform" },
      );
      tokenStore.set(res.accessToken, res.refreshToken, "platform");
      set({ user: toSession(res.user) });
      return { ok: true };
    } catch (e) {
      // Surface the backend's actual message (e.g. distinguishes wrong credentials from an
      // inactive superadmin account) instead of a generic fallback that swallows it.
      return {
        ok: false,
        error: e instanceof ApiError ? e.message : "Email atau password salah. Coba lagi.",
      };
    }
  },

  logout: async () => {
    const refreshToken = tokenStore.refresh("platform");
    if (refreshToken) {
      try {
        await api.post(
          endpoints.auth.logout,
          { refreshToken },
          { auth: false, tokenDomain: "platform" },
        );
      } catch {
        /* best-effort */
      }
    }
    tokenStore.clear("platform");
    set({ user: null });
  },
}));
