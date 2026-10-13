import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  deleteAccount as deleteAccountRequest,
  login as loginRequest,
  logout as logoutRequest,
  me,
  register as registerRequest,
  type AuthSession
} from "../api/auth";
import { isApiError } from "../api/client";
import type { AuthUser } from "../api/types";
import { resetFeedback } from "../shared/feedback";
import { clearTokens, loadTokens, peekTokens, saveTokens, subscribeTokens, type AuthTokens } from "../store/tokenStore";

type AuthContextValue = {
  user: AuthUser | null;
  isBooting: boolean;
  isAuthenticated: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
};

const CURRENT_USER_KEY = ["auth", "current-user"] as const;
const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [tokens, setTokens] = useState<AuthTokens | null>(null);
  const [bootstrapping, setBootstrapping] = useState(true);

  // The token store is the single source of truth: silent refreshes and forced
  // sign-outs performed by the API client reach the UI through this listener.
  useEffect(() => {
    let active = true;
    const unsubscribe = subscribeTokens((next) => {
      if (active) {
        setTokens(next);
      }
    });
    loadTokens()
      .then((restored) => active && setTokens(restored))
      .finally(() => active && setBootstrapping(false));
    return () => {
      active = false;
      unsubscribe();
    };
  }, []);

  const isSignedIn = Boolean(tokens?.refreshToken);

  const userQuery = useQuery<AuthUser, Error>({
    queryKey: CURRENT_USER_KEY,
    enabled: !bootstrapping && isSignedIn,
    queryFn: me,
    retry: false,
    staleTime: Infinity
  });

  // A definitive rejection of the session (e.g. deleted or suspended account) ends it.
  useEffect(() => {
    if (userQuery.isError && isApiError(userQuery.error) && (userQuery.error.status === 401 || userQuery.error.status === 403)) {
      void clearTokens();
    }
  }, [userQuery.isError, userQuery.error]);

  useEffect(() => {
    if (!bootstrapping && !isSignedIn) {
      queryClient.clear();
      resetFeedback();
    }
  }, [bootstrapping, isSignedIn, queryClient]);

  const applySession = useCallback(
    async (session: AuthSession) => {
      queryClient.clear();
      queryClient.setQueryData(CURRENT_USER_KEY, session.user);
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    },
    [queryClient]
  );

  const signIn = useCallback(async (email: string, password: string) => applySession(await loginRequest(email, password)), [applySession]);
  const signUp = useCallback(
    async (email: string, password: string) => applySession(await registerRequest(email, password)),
    [applySession]
  );

  const signOut = useCallback(async () => {
    const refreshToken = peekTokens()?.refreshToken;
    try {
      if (refreshToken) {
        await logoutRequest(refreshToken);
      }
    } catch {
      // Best effort: local cleanup always wins.
    }
    await clearTokens();
  }, []);

  const deleteAccount = useCallback(async (password: string) => {
    await deleteAccountRequest(password);
    await clearTokens();
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      user: isSignedIn ? (userQuery.data ?? null) : null,
      isBooting: bootstrapping || (isSignedIn && userQuery.isLoading),
      isAuthenticated: isSignedIn && Boolean(userQuery.data),
      signIn,
      signUp,
      signOut,
      deleteAccount
    }),
    [bootstrapping, deleteAccount, isSignedIn, signIn, signOut, signUp, userQuery.data, userQuery.isLoading]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
