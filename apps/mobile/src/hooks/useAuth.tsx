import { useQueryClient } from "@tanstack/react-query";
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import * as authApi from "../api/auth";
import { api, ApiError, NetworkError } from "../api/client";
import type { AuthUser } from "../api/types";
import { tokenStore } from "../store/tokenStore";

/** "unreachable": a session exists but the server could not be contacted at launch (offline). */
export type AuthStatus = "booting" | "authenticated" | "anonymous" | "unreachable";

type AuthContextValue = {
  status: AuthStatus;
  user: AuthUser | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
  retryBoot: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [bootAttempt, setBootAttempt] = useState(0);

  const signOutLocally = useCallback(() => {
    setUser(null);
    setStatus("anonymous");
    queryClient.clear(); // never keep one person's data in memory for the next account
  }, [queryClient]);

  useEffect(() => {
    api.setSessionExpiredHandler(signOutLocally);
  }, [signOutLocally]);

  useEffect(() => {
    const controller = new AbortController();
    (async () => {
      setStatus("booting");
      if (!(await tokenStore.load())) {
        setStatus("anonymous");
        return;
      }
      try {
        const me = await authApi.fetchMe(controller.signal);
        setUser(me);
        setStatus("authenticated");
      } catch (error) {
        if (controller.signal.aborted) return;
        if (error instanceof NetworkError) setStatus("unreachable");
        else if (error instanceof ApiError && error.status === 401) signOutLocally();
        else setStatus("unreachable");
      }
    })();
    return () => controller.abort();
  }, [bootAttempt, signOutLocally]);

  const login = useCallback(
    async (email: string, password: string) => {
      const session = await authApi.login(email, password);
      queryClient.clear();
      setUser(session.user);
      setStatus("authenticated");
    },
    [queryClient]
  );

  const register = useCallback(
    async (email: string, password: string) => {
      const session = await authApi.register(email, password);
      queryClient.clear();
      setUser(session.user);
      setStatus("authenticated");
    },
    [queryClient]
  );

  const logout = useCallback(async () => {
    await authApi.logout();
    signOutLocally();
  }, [signOutLocally]);

  const deleteAccount = useCallback(
    async (password: string) => {
      await authApi.deleteAccount(password);
      signOutLocally();
    },
    [signOutLocally]
  );

  const retryBoot = useCallback(() => setBootAttempt((n) => n + 1), []);

  const value = useMemo<AuthContextValue>(
    () => ({ status, user, login, register, logout, deleteAccount, retryBoot }),
    [status, user, login, register, logout, deleteAccount, retryBoot]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}

/** The signed-in user; only call from screens rendered when authenticated. */
export function useCurrentUser(): AuthUser {
  const { user } = useAuth();
  if (!user) throw new Error("useCurrentUser requires an authenticated session");
  return user;
}
