import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as authApi from "../api/auth";
import { setSessionBridge } from "../api/client";
import { clearTokens, loadTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

type AuthContextValue = {
  status: "booting" | "signedOut" | "signedIn";
  user: authApi.AuthUser | null;
  getAccessToken: () => string | null;
  refresh: () => Promise<string | null>;
  applySession: (session: authApi.AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Local cleanup after the account was deleted server-side. */
  forgetSession: () => Promise<void>;
};

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000, refetchOnWindowFocus: false },
    mutations: { retry: 0 }
  }
});

const AuthContext = createContext<AuthContextValue | null>(null);

function AuthState({ children }: { children: React.ReactNode }) {
  const tokensRef = useRef<AuthTokens | null>(null);
  const [status, setStatus] = useState<AuthContextValue["status"]>("booting");
  const [user, setUser] = useState<authApi.AuthUser | null>(null);

  const forgetSession = useCallback(async () => {
    tokensRef.current = null;
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, []);

  const applySession = useCallback(async (session: authApi.AuthSession) => {
    tokensRef.current = { accessToken: session.accessToken, refreshToken: session.refreshToken };
    await saveTokens(tokensRef.current);
    setUser(session.user);
    setStatus("signedIn");
  }, []);

  const refresh = useCallback(async (): Promise<string | null> => {
    const current = tokensRef.current;
    if (!current?.refreshToken) {
      return null;
    }
    try {
      const next = await authApi.refreshSession(current.refreshToken);
      await applySession(next);
      return next.accessToken;
    } catch {
      await forgetSession();
      return null;
    }
  }, [applySession, forgetSession]);

  const signOut = useCallback(async () => {
    const refreshToken = tokensRef.current?.refreshToken;
    await forgetSession();
    if (refreshToken) {
      authApi.logout(refreshToken).catch(() => undefined);
    }
  }, [forgetSession]);

  const getAccessToken = useCallback(() => tokensRef.current?.accessToken ?? null, []);

  useEffect(() => {
    setSessionBridge({ getAccessToken, refresh });
  }, [getAccessToken, refresh]);

  useEffect(() => {
    let active = true;
    (async () => {
      const restored = await loadTokens();
      if (!active) {
        return;
      }
      if (!restored) {
        setStatus("signedOut");
        return;
      }
      tokensRef.current = restored;
      setSessionBridge({ getAccessToken, refresh });
      try {
        const current = await authApi.me();
        if (active) {
          setUser(current);
          setStatus("signedIn");
        }
      } catch {
        if (active) {
          await forgetSession();
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [forgetSession, getAccessToken, refresh]);

  const value = useMemo<AuthContextValue>(
    () => ({ status, user, getAccessToken, refresh, applySession, signOut, forgetSession }),
    [applySession, forgetSession, getAccessToken, refresh, signOut, status, user]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthState>{children}</AuthState>
    </QueryClientProvider>
  );
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
