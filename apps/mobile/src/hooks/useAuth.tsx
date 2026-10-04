import React, { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import * as authApi from "../api/auth";
import { api, isApiError } from "../api/client";
import type { Me } from "../api/types";
import { tokenStore } from "../store/tokenStore";

export type AuthStatus = "booting" | "signedOut" | "signedIn" | "bootError";

type AuthContextValue = {
  status: AuthStatus;
  user: Me | null;
  /** Set when the stored session could not be verified because the server was unreachable. */
  bootError: unknown;
  retryBoot: () => void;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  /** Clears the session after the account has been deleted server side. */
  finishAccountDeletion: () => void;
  updateUser: (user: Me) => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<Me | null>(null);
  const [bootError, setBootError] = useState<unknown>(null);
  const [bootAttempt, setBootAttempt] = useState(0);

  const signOutLocally = useCallback(() => {
    setUser(null);
    setStatus("signedOut");
    queryClient.clear();
  }, [queryClient]);

  // The API client tells us when the session can no longer be refreshed.
  useEffect(() => {
    api.setAuthLostHandler(signOutLocally);
    return () => api.setAuthLostHandler(undefined);
  }, [signOutLocally]);

  // Restore a persisted session.
  useEffect(() => {
    let active = true;
    setStatus("booting");
    (async () => {
      const tokens = await tokenStore.load();
      if (!tokens) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.getMe();
        if (!active) return;
        setUser(me);
        setBootError(null);
        setStatus("signedIn");
      } catch (error) {
        if (!active) return;
        if (isApiError(error) && (error.status === 401 || error.status === 403)) {
          await tokenStore.clear();
          setStatus("signedOut");
        } else {
          setBootError(error);
          setStatus("bootError");
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [bootAttempt]);

  const login = useCallback(
    async (email: string, password: string) => {
      const data = await authApi.login(email, password);
      queryClient.clear();
      setUser(data.user);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const register = useCallback(
    async (email: string, password: string) => {
      const data = await authApi.register(email, password);
      queryClient.clear();
      setUser(data.user);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const logout = useCallback(async () => {
    await authApi.logout();
    signOutLocally();
  }, [signOutLocally]);

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      user,
      bootError,
      retryBoot: () => setBootAttempt((n) => n + 1),
      login,
      register,
      logout,
      finishAccountDeletion: signOutLocally,
      updateUser: setUser
    }),
    [status, user, bootError, login, register, logout, signOutLocally]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside <AuthProvider>");
  return value;
}

export function useCurrentUserId(): string {
  const { user } = useAuth();
  if (!user) throw new Error("No signed-in user");
  return user.id;
}
