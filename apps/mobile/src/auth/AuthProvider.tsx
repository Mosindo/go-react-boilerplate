import React, { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "../api/endpoints";
import { isApiError, onSessionExpired } from "../api/client";
import { getTokens, restoreTokens, setTokens } from "../api/session";
import type { AuthResponse, AuthUser } from "../api/types";
import { showToast } from "../ui/Toast";

type AuthState = { status: "booting" } | { status: "guest" } | { status: "authenticated"; user: AuthUser };

type AuthContextValue = {
  state: AuthState;
  signIn: (response: AuthResponse) => Promise<void>;
  signOut: (options?: { remote?: boolean }) => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [state, setState] = useState<AuthState>({ status: "booting" });

  useEffect(() => {
    let active = true;
    (async () => {
      const restored = await restoreTokens();
      if (!restored) {
        if (active) setState({ status: "guest" });
        return;
      }
      try {
        const user = await authApi.me();
        if (active) setState({ status: "authenticated", user });
      } catch (error) {
        if (isApiError(error) && error.status === 401) {
          await setTokens(null);
        }
        if (active) setState({ status: "guest" });
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    onSessionExpired(() => {
      queryClient.clear();
      setState({ status: "guest" });
      showToast("Ta session a expiré. Reconnecte-toi.", "info");
    });
  }, [queryClient]);

  const signIn = useCallback(
    async (response: AuthResponse) => {
      queryClient.clear();
      await setTokens({ accessToken: response.accessToken, refreshToken: response.refreshToken });
      setState({ status: "authenticated", user: response.user });
    },
    [queryClient]
  );

  const signOut = useCallback(
    async ({ remote = true }: { remote?: boolean } = {}) => {
      const current = getTokens();
      if (remote && current?.refreshToken) {
        try {
          await authApi.logout(current.refreshToken);
        } catch {
          // Best effort: the local session is cleared anyway.
        }
      }
      await setTokens(null);
      queryClient.clear();
      setState({ status: "guest" });
    },
    [queryClient]
  );

  const value = useMemo(() => ({ state, signIn, signOut }), [state, signIn, signOut]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}

export function useCurrentUser(): AuthUser {
  const { state } = useAuth();
  if (state.status !== "authenticated") throw new Error("useCurrentUser requires an authenticated session");
  return state.user;
}
