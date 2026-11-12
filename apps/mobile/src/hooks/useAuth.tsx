import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError, setRefreshHandler } from "../api/client";
import { authApi } from "../api/endpoints";
import type { AuthSession, AuthUser } from "../api/types";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";

type AuthStatus = "booting" | "signedOut" | "signedIn" | "bootError";

type AuthContextValue = {
  status: AuthStatus;
  user: AuthUser | null;
  /** Bumped whenever the access token changes (used to re-authenticate the realtime socket). */
  tokenVersion: number;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: (options?: { remote?: boolean }) => Promise<void>;
  retryBoot: () => void;
};

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (count, error) => !(error instanceof ApiError && error.status < 500) && count < 2,
      staleTime: 30_000,
      refetchOnWindowFocus: false
    },
    mutations: { retry: 0 }
  }
});

const AuthContext = createContext<AuthContextValue | null>(null);

function AuthProviderInner({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [tokenVersion, setTokenVersion] = useState(0);
  const [bootAttempt, setBootAttempt] = useState(0);
  const signingOut = useRef(false);

  const clearLocal = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, []);

  const signIn = useCallback(async (session: AuthSession) => {
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    queryClient.clear();
    setUser(session.user);
    setTokenVersion((v) => v + 1);
    setStatus("signedIn");
  }, []);

  const signOut = useCallback(
    async ({ remote = true }: { remote?: boolean } = {}) => {
      if (signingOut.current) return;
      signingOut.current = true;
      try {
        const tokens = await getTokens();
        if (remote && tokens?.refreshToken) {
          await authApi.logout(tokens.refreshToken).catch(() => undefined);
        }
      } finally {
        await clearLocal();
        signingOut.current = false;
      }
    },
    [clearLocal]
  );

  // Silent refresh used by the API client on 401 (single-flight there).
  useEffect(() => {
    setRefreshHandler(async () => {
      const tokens = await getTokens();
      if (!tokens?.refreshToken) {
        return null;
      }
      try {
        const next = await authApi.refresh(tokens.refreshToken);
        await saveTokens({ accessToken: next.accessToken, refreshToken: next.refreshToken });
        setUser(next.user);
        setTokenVersion((v) => v + 1);
        return next.accessToken;
      } catch (error) {
        if (error instanceof ApiError) {
          await clearLocal();
        }
        return null;
      }
    });
    return () => setRefreshHandler(null);
  }, [clearLocal]);

  useEffect(() => {
    let active = true;
    (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.me();
        if (!active) return;
        setUser(me);
        setStatus("signedIn");
      } catch (error) {
        if (!active) return;
        if (error instanceof ApiError && error.status === 401) {
          await clearLocal();
        } else {
          setStatus("bootError");
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [bootAttempt, clearLocal]);

  const value = useMemo<AuthContextValue>(
    () => ({ status, user, tokenVersion, signIn, signOut, retryBoot: () => setBootAttempt((n) => n + 1) }),
    [status, user, tokenVersion, signIn, signOut]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProviderInner>{children}</AuthProviderInner>
    </QueryClientProvider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
