import { useQueryClient } from "@tanstack/react-query";
import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import * as authApi from "../api/auth";
import { ApiError } from "../api/client";
import {
  clearTokens,
  getTokensSync,
  loadTokens,
  saveTokens,
  subscribeToSession
} from "../api/session";
import type { AuthSession, Me } from "../api/types";

type AuthState =
  | { status: "loading" }
  | { status: "signedOut" }
  | { status: "offline"; message: string }
  | { status: "signedIn"; me: Me };

type AuthApi = {
  state: AuthState;
  /** Stores a freshly obtained session (login / register). */
  startSession: (session: AuthSession) => Promise<void>;
  /** Re-reads /me, e.g. after finishing an onboarding step. */
  refreshMe: () => Promise<Me | null>;
  retry: () => void;
  signOut: () => Promise<void>;
  /** Clears local state after the server deleted the account. */
  afterAccountDeleted: () => Promise<void>;
};

const AuthContext = createContext<AuthApi | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [state, setState] = useState<AuthState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let active = true;
    async function boot() {
      const tokens = await loadTokens();
      if (!tokens) {
        if (active) setState({ status: "signedOut" });
        return;
      }
      try {
        const me = await authApi.fetchMe();
        if (active) setState({ status: "signedIn", me });
      } catch (error) {
        if (!active) return;
        if (error instanceof ApiError && error.status === 401) {
          await clearTokens();
          setState({ status: "signedOut" });
        } else {
          setState({
            status: "offline",
            message: error instanceof Error ? error.message : "Connexion impossible."
          });
        }
      }
    }
    setState({ status: "loading" });
    void boot();
    return () => {
      active = false;
    };
  }, [attempt]);

  // The API client clears tokens when the refresh token is rejected: sign out everywhere.
  useEffect(
    () =>
      subscribeToSession((tokens) => {
        if (!tokens) {
          queryClient.clear();
          setState((current) =>
            current.status === "signedIn" ? { status: "signedOut" } : current
          );
        }
      }),
    [queryClient]
  );

  const startSession = useCallback(
    async (session: AuthSession) => {
      queryClient.clear();
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      setState({ status: "signedIn", me: session.user });
    },
    [queryClient]
  );

  const refreshMe = useCallback(async () => {
    try {
      const me = await authApi.fetchMe();
      setState({ status: "signedIn", me });
      return me;
    } catch {
      return null;
    }
  }, []);

  const signOut = useCallback(async () => {
    const refreshToken = getTokensSync()?.refreshToken;
    if (refreshToken) {
      try {
        await authApi.logout(refreshToken);
      } catch {
        // Best effort: the local session is dropped regardless.
      }
    }
    await clearTokens();
    queryClient.clear();
    setState({ status: "signedOut" });
  }, [queryClient]);

  const afterAccountDeleted = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setState({ status: "signedOut" });
  }, [queryClient]);

  const retry = useCallback(() => setAttempt((n) => n + 1), []);

  const value = useMemo<AuthApi>(
    () => ({ state, startSession, refreshMe, retry, signOut, afterAccountDeleted }),
    [state, startSession, refreshMe, retry, signOut, afterAccountDeleted]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthApi {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return context;
}

/** The signed-in account; only call below the authenticated navigator. */
export function useMe(): Me {
  const { state } = useAuth();
  if (state.status !== "signedIn") {
    throw new Error("useMe requires a signed-in user");
  }
  return state.me;
}
