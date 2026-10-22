import React, { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import * as authApi from "../api/auth";
import { setSessionExpiredHandler } from "../api/client";
import type { AuthResponse, User } from "../api/types";
import { clearTokens, getTokens, saveTokens, subscribeTokens } from "../store/tokenStore";

type AuthContextValue = {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isBooting: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
  /** Re-reads /me (e.g. after the profile becomes complete). */
  refreshUser: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [user, setUser] = useState<User | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [isBooting, setIsBooting] = useState(true);

  const reset = useCallback(() => {
    setUser(null);
    setAccessToken(null);
    queryClient.clear();
  }, [queryClient]);

  useEffect(() => subscribeTokens((tokens) => setAccessToken(tokens?.accessToken ?? null)), []);

  useEffect(() => {
    setSessionExpiredHandler(reset);
    return () => setSessionExpiredHandler(null);
  }, [reset]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const tokens = await getTokens();
        if (!tokens) {
          return;
        }
        setAccessToken(tokens.accessToken);
        const me = await authApi.fetchMe();
        if (!cancelled) {
          setUser(me);
        }
      } catch {
        // Offline or expired session: show the sign-in screen rather than a broken app.
        if (!cancelled) {
          setAccessToken(null);
        }
      } finally {
        if (!cancelled) {
          setIsBooting(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const adopt = useCallback(async (response: AuthResponse) => {
    await saveTokens({ accessToken: response.accessToken, refreshToken: response.refreshToken });
    setUser(response.user);
  }, []);

  const signIn = useCallback(
    async (email: string, password: string) => {
      queryClient.clear();
      await adopt(await authApi.login(email, password));
    },
    [adopt, queryClient]
  );

  const signUp = useCallback(
    async (email: string, password: string) => {
      queryClient.clear();
      await adopt(await authApi.register(email, password));
    },
    [adopt, queryClient]
  );

  const signOut = useCallback(async () => {
    const tokens = await getTokens();
    if (tokens?.refreshToken) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // The local session is cleared either way.
      }
    }
    await clearTokens();
    reset();
  }, [reset]);

  const deleteAccount = useCallback(
    async (password: string) => {
      await authApi.deleteAccount(password);
      await clearTokens();
      reset();
    },
    [reset]
  );

  const refreshUser = useCallback(async () => {
    setUser(await authApi.fetchMe());
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      accessToken,
      isAuthenticated: user !== null && accessToken !== null,
      isBooting,
      signIn,
      signUp,
      signOut,
      deleteAccount,
      refreshUser
    }),
    [user, accessToken, isBooting, signIn, signUp, signOut, deleteAccount, refreshUser]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return context;
}
