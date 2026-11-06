import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "../api/services";
import { ApiError, setSessionExpiredHandler } from "../api/client";
import { clearTokens, currentTokens, loadTokens, saveTokens } from "../api/tokenStore";
import type { AuthResponse, AuthUser } from "../api/types";

type Status = "booting" | "anonymous" | "authenticated";

type AuthContextValue = {
  status: Status;
  user: AuthUser | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
  /** Drops local credentials after the server already invalidated the session. */
  forgetSession: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  const forgetSession = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("anonymous");
  }, [queryClient]);

  useEffect(() => {
    setSessionExpiredHandler(() => {
      void forgetSession();
    });
  }, [forgetSession]);

  useEffect(() => {
    let alive = true;
    (async () => {
      const tokens = await loadTokens();
      if (!tokens) {
        if (alive) setStatus("anonymous");
        return;
      }
      try {
        const me = await authApi.me();
        if (!alive) return;
        setUser(me);
        setStatus("authenticated");
      } catch (error) {
        if (!alive) return;
        // A network failure at boot must not log the user out: keep tokens, show login.
        if (!(error instanceof ApiError && error.isNetwork)) await clearTokens();
        setStatus("anonymous");
      }
    })();
    return () => {
      alive = false;
    };
  }, []);

  const adopt = useCallback(
    async (res: AuthResponse) => {
      queryClient.clear();
      await saveTokens({ accessToken: res.accessToken, refreshToken: res.refreshToken });
      setUser(res.user);
      setStatus("authenticated");
    },
    [queryClient]
  );

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      user,
      login: async (email, password) => adopt(await authApi.login(email.trim(), password)),
      register: async (email, password) => adopt(await authApi.register(email.trim(), password)),
      logout: async () => {
        const refresh = currentTokens()?.refreshToken;
        if (refresh) await authApi.logout(refresh).catch(() => undefined);
        await forgetSession();
      },
      deleteAccount: async (password) => {
        await authApi.deleteAccount(password);
        await forgetSession();
      },
      forgetSession
    }),
    [status, user, adopt, forgetSession]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
