import { useQueryClient } from "@tanstack/react-query";
import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import * as authApi from "../api/auth";
import { ApiError, onSessionExpired } from "../api/client";
import type { AuthResponse, Me } from "../api/types";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";

type Status = "booting" | "signedOut" | "signedIn" | "unreachable";

type AuthContextValue = {
  status: Status;
  user: Me | null;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string, birthDate: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Re-reads /me (after the profile or photos changed). */
  refreshUser: () => Promise<Me | null>;
  /** Clears local state after the account was deleted server-side. */
  accountDeleted: () => Promise<void>;
  retryBoot: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<Me | null>(null);
  const [bootNonce, setBootNonce] = useState(0);

  const reset = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, [queryClient]);

  useEffect(() => {
    onSessionExpired(() => {
      queryClient.clear();
      setUser(null);
      setStatus("signedOut");
    });
    return () => onSessionExpired(null);
  }, [queryClient]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        if (!cancelled) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.fetchMe();
        if (cancelled) return;
        setUser(me);
        setStatus("signedIn");
      } catch (err) {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 0) setStatus("unreachable");
        else await reset();
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [bootNonce, reset]);

  const applySession = useCallback(
    async (session: AuthResponse) => {
      queryClient.clear();
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      setUser(session.user);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const signIn = useCallback(async (email: string, password: string) => applySession(await authApi.login(email, password)), [applySession]);
  const signUp = useCallback(
    async (email: string, password: string, birthDate: string) => applySession(await authApi.register(email, password, birthDate)),
    [applySession]
  );

  const signOut = useCallback(async () => {
    const tokens = await getTokens();
    if (tokens) await authApi.logout(tokens.refreshToken).catch(() => undefined);
    await reset();
  }, [reset]);

  const refreshUser = useCallback(async () => {
    try {
      const me = await authApi.fetchMe();
      setUser(me);
      return me;
    } catch {
      return null;
    }
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      user,
      signIn,
      signUp,
      signOut,
      refreshUser,
      accountDeleted: reset,
      retryBoot: () => {
        setStatus("booting");
        setBootNonce((n) => n + 1);
      }
    }),
    [status, user, signIn, signUp, signOut, refreshUser, reset]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside <AuthProvider>");
  return ctx;
}

/** The signed-in user; only call below the auth gate. */
export function useCurrentUser(): Me {
  const { user } = useAuth();
  if (!user) throw new Error("useCurrentUser requires an authenticated user");
  return user;
}
