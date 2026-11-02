import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "../api/endpoints";
import { ApiError, refreshSessionNow, setSessionListener } from "../api/client";
import type { AuthSession, AuthUser } from "../api/types";
import { clearTokens, getCachedTokens, loadTokens, saveTokens } from "../store/tokenStore";

type Status = "loading" | "out" | "in" | "unreachable";

type AuthValue = {
  status: Status;
  user: AuthUser | null;
  accessToken: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
  retryBoot: () => void;
};

const AuthContext = createContext<AuthValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("loading");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [bootAttempt, setBootAttempt] = useState(0);

  const reset = useCallback(() => {
    setUser(null);
    setAccessToken(null);
    setStatus("out");
    queryClient.clear();
  }, [queryClient]);

  useEffect(() => {
    setSessionListener({
      onTokens: (tokens) => setAccessToken(tokens.accessToken),
      onExpired: reset
    });
    return () => setSessionListener(null);
  }, [reset]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setStatus("loading");
      const tokens = await loadTokens();
      if (!tokens) {
        if (!cancelled) setStatus("out");
        return;
      }
      try {
        const me = await authApi.me();
        if (cancelled) return;
        setUser(me);
        setAccessToken(getCachedTokens()?.accessToken ?? tokens.accessToken);
        setStatus("in");
      } catch (error) {
        if (cancelled) return;
        if (error instanceof ApiError && error.status === 0) {
          setStatus("unreachable");
        } else {
          await clearTokens();
          setStatus("out");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [bootAttempt]);

  // Keep the access token fresh so authenticated image requests keep working.
  useEffect(() => {
    if (status !== "in") return;
    const id = setInterval(() => void refreshSessionNow(), 10 * 60 * 1000);
    return () => clearInterval(id);
  }, [status]);

  const startSession = useCallback(async (session: AuthSession) => {
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    setUser(session.user);
    setAccessToken(session.accessToken);
    setStatus("in");
  }, []);

  const value = useMemo<AuthValue>(
    () => ({
      status,
      user,
      accessToken,
      login: async (email, password) => startSession(await authApi.login(email, password)),
      register: async (email, password) => startSession(await authApi.register(email, password)),
      logout: async () => {
        const refresh = getCachedTokens()?.refreshToken;
        if (refresh) await authApi.logout(refresh).catch(() => undefined);
        await clearTokens();
        reset();
      },
      deleteAccount: async (password) => {
        await authApi.deleteAccount(password);
        await clearTokens();
        reset();
      },
      retryBoot: () => setBootAttempt((n) => n + 1)
    }),
    [status, user, accessToken, startSession, reset]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
