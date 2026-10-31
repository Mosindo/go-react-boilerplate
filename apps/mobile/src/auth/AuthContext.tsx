import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import * as authApi from "../api/auth";
import { ApiError, ensureFreshAccessToken, setSessionExpiredHandler } from "../api/client";
import type { AuthSession, AuthUser } from "../api/types";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";

type Status = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: Status;
  user: AuthUser | null;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state after the server already deleted the account. */
  dropSession: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);
const TOKEN_KEEPALIVE_MS = 10 * 60 * 1000;

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  const dropSession = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, [queryClient]);

  useEffect(() => {
    setSessionExpiredHandler(() => {
      queryClient.clear();
      setUser(null);
      setStatus("signedOut");
    });
    return () => setSessionExpiredHandler(null);
  }, [queryClient]);

  // Restore a stored session on launch.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        if (!cancelled) setStatus("signedOut");
        return;
      }
      try {
        const current = await authApi.me();
        if (!cancelled) {
          setUser(current);
          setStatus("signedIn");
        }
      } catch (error) {
        // Only a definitive auth failure ends the session; network errors keep it for retry.
        if (error instanceof ApiError && error.status === 401) {
          await clearTokens();
        }
        if (!cancelled) setStatus("signedOut");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Keep the access token fresh: photos are fetched with it outside of apiRequest.
  useEffect(() => {
    if (status !== "signedIn") return;
    const timer = setInterval(() => {
      void ensureFreshAccessToken();
    }, TOKEN_KEEPALIVE_MS);
    return () => clearInterval(timer);
  }, [status]);

  const signIn = useCallback(
    async (session: AuthSession) => {
      queryClient.clear();
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      setUser(session.user);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const signOut = useCallback(async () => {
    const tokens = await getTokens();
    if (tokens) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // Logging out locally must always succeed, even offline.
      }
    }
    await dropSession();
  }, [dropSession]);

  const value = useMemo(() => ({ status, user, signIn, signOut, dropSession }), [status, user, signIn, signOut, dropSession]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return ctx;
}
