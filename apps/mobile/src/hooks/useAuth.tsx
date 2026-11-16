import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { QueryClient, QueryClientProvider, useQueryClient } from "@tanstack/react-query";
import { authApi } from "../api/auth";
import { ApiError, configureSession } from "../api/client";
import type { AuthSession, AuthUser } from "../api/types";
import { clearTokens, loadTokens, saveTokens, type StoredTokens } from "../store/tokenStore";

type AuthStatus = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: AuthStatus;
  user: AuthUser | null;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state only (account deleted, session revoked elsewhere). */
  forgetSession: () => Promise<void>;
  setUser: (user: AuthUser) => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnWindowFocus: false,
      retry: (count, error) => !(error instanceof ApiError && error.status < 500) && count < 2
    },
    mutations: { retry: 0 }
  }
});

function AuthProviderInner({ children }: { children: React.ReactNode }) {
  const client = useQueryClient();
  const tokensRef = useRef<StoredTokens | null>(null);
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  const forgetSession = useCallback(async () => {
    tokensRef.current = null;
    await clearTokens();
    client.clear();
    setUser(null);
    setStatus("signedOut");
  }, [client]);

  const refresh = useCallback(async (): Promise<string | null> => {
    const current = tokensRef.current;
    if (!current?.refreshToken) {
      return null;
    }
    try {
      const session = await authApi.refresh(current.refreshToken);
      tokensRef.current = { accessToken: session.accessToken, refreshToken: session.refreshToken };
      await saveTokens(tokensRef.current);
      setUser(session.user);
      return session.accessToken;
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        await forgetSession();
      }
      return null;
    }
  }, [forgetSession]);

  useEffect(() => {
    configureSession({ getAccessToken: () => tokensRef.current?.accessToken ?? null, refresh });
  }, [refresh]);

  useEffect(() => {
    let active = true;
    (async () => {
      const stored = await loadTokens();
      if (!active) {
        return;
      }
      if (!stored) {
        setStatus("signedOut");
        return;
      }
      tokensRef.current = stored;
      try {
        const me = await authApi.me();
        if (active) {
          setUser(me);
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
  }, [forgetSession]);

  const signIn = useCallback(
    async (session: AuthSession) => {
      tokensRef.current = { accessToken: session.accessToken, refreshToken: session.refreshToken };
      await saveTokens(tokensRef.current);
      client.clear();
      setUser(session.user);
      setStatus("signedIn");
    },
    [client]
  );

  const signOut = useCallback(async () => {
    const refreshToken = tokensRef.current?.refreshToken;
    if (refreshToken) {
      try {
        await authApi.logout(refreshToken);
      } catch {
        // Best effort: local sign-out always wins.
      }
    }
    await forgetSession();
  }, [forgetSession]);

  const value = useMemo<AuthContextValue>(() => ({ status, user, signIn, signOut, forgetSession, setUser }), [status, user, signIn, signOut, forgetSession]);
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
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return ctx;
}
