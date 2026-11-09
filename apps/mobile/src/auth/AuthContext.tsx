import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "../api/endpoints";
import { setSessionExpiredHandler } from "../api/client";
import type { AuthResponse, AuthUser } from "../api/types";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import { request } from "../api/client";

type Status = "loading" | "signedOut" | "signedIn";

type AuthValue = {
  status: Status;
  user: AuthUser | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
};

const AuthContext = createContext<AuthValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("loading");
  const [user, setUser] = useState<AuthUser | null>(null);

  const signOutLocally = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, [queryClient]);

  const accept = useCallback(async (session: AuthResponse) => {
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    setUser(session.user);
    setStatus("signedIn");
  }, []);

  // Restore the session on launch; /me also validates (and refreshes) the stored tokens.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        if (!cancelled) setStatus("signedOut");
        return;
      }
      try {
        const me = await request<AuthUser>("/me");
        if (!cancelled) {
          setUser(me);
          setStatus("signedIn");
        }
      } catch (error) {
        const code = (error as { status?: number }).status;
        if (cancelled) return;
        if (code === 401) {
          await signOutLocally();
        } else {
          // Offline at launch: keep the stored session and let screens show their own retry state.
          setStatus("signedIn");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [signOutLocally]);

  useEffect(() => {
    setSessionExpiredHandler(() => {
      queryClient.clear();
      setUser(null);
      setStatus("signedOut");
    });
    return () => setSessionExpiredHandler(null);
  }, [queryClient]);

  const value = useMemo<AuthValue>(
    () => ({
      status,
      user,
      login: async (email, password) => accept(await authApi.login(email.trim(), password)),
      register: async (email, password) => accept(await authApi.register(email.trim(), password)),
      logout: async () => {
        const tokens = await getTokens();
        if (tokens) {
          await authApi.logout(tokens.refreshToken).catch(() => undefined);
        }
        await signOutLocally();
      },
      deleteAccount: async (password) => {
        await authApi.deleteAccount(password);
        await signOutLocally();
      }
    }),
    [status, user, accept, signOutLocally]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return ctx;
}
