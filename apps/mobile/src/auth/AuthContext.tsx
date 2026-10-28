import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import * as authApi from "../api/auth";
import { ApiError, setSessionExpiredListener } from "../api/client";
import { keys } from "../api/keys";
import type { Account, AuthResponse } from "../api/types";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";

type AuthContextValue = {
  user: Account | null;
  isBooting: boolean;
  /** Tokens exist but the account could not be loaded (network/server error). */
  bootError: boolean;
  retryBoot: () => void;
  signIn: (session: AuthResponse) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local session without calling the API (account deleted / expired). */
  dropSession: () => Promise<void>;
  refreshAccount: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [hasTokens, setHasTokens] = useState(false);
  const [bootstrapped, setBootstrapped] = useState(false);

  const dropSession = useCallback(async () => {
    await clearTokens();
    setHasTokens(false);
    queryClient.clear();
  }, [queryClient]);

  useEffect(() => {
    let active = true;
    void getTokens().then((tokens) => {
      if (active) {
        setHasTokens(Boolean(tokens));
        setBootstrapped(true);
      }
    });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    setSessionExpiredListener(() => {
      setHasTokens(false);
      queryClient.clear();
    });
    return () => setSessionExpiredListener(null);
  }, [queryClient]);

  const meQuery = useQuery<Account, Error>({
    queryKey: keys.me,
    queryFn: authApi.fetchMe,
    enabled: bootstrapped && hasTokens,
    retry: (count, error) => !(error instanceof ApiError && error.status === 401) && count < 2
  });

  useEffect(() => {
    if (meQuery.error instanceof ApiError && meQuery.error.status === 401) {
      void dropSession();
    }
  }, [meQuery.error, dropSession]);

  const signIn = useCallback(
    async (session: AuthResponse) => {
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      queryClient.setQueryData(keys.me, session.user);
      setHasTokens(true);
    },
    [queryClient]
  );

  const signOut = useCallback(async () => {
    const tokens = await getTokens();
    if (tokens?.refreshToken) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // best effort; local cleanup always wins
      }
    }
    await dropSession();
  }, [dropSession]);

  const refreshAccount = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: keys.me });
  }, [queryClient]);

  const value = useMemo<AuthContextValue>(
    () => ({
      user: hasTokens ? (meQuery.data ?? null) : null,
      isBooting: !bootstrapped || (hasTokens && meQuery.isPending),
      bootError: hasTokens && meQuery.isError,
      retryBoot: () => void meQuery.refetch(),
      signIn,
      signOut,
      dropSession,
      refreshAccount
    }),
    [bootstrapped, dropSession, hasTokens, meQuery, refreshAccount, signIn, signOut]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
