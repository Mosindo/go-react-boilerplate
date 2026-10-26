import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi, type AuthResponse, type AuthUser } from "../api";
import { api, isApiError } from "../api/client";
import { session } from "./session";
import { loadTokens } from "./tokenStore";

type Status = "booting" | "signedOut" | "signedIn" | "failed";

type AuthContextValue = {
  status: Status;
  user: AuthUser | null;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state after the server deleted the account. */
  afterAccountDeleted: () => Promise<void>;
  retryBoot: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [bootAttempt, setBootAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const tokens = await loadTokens();
      if (!tokens) {
        if (!cancelled) setStatus("signedOut");
        return;
      }
      session.restore(tokens);
      try {
        const me = await api<AuthUser>("/me");
        if (!cancelled) {
          setUser(me);
          setStatus("signedIn");
        }
      } catch (err) {
        if (cancelled) return;
        if (isApiError(err, 401) || isApiError(err, 403)) {
          await session.clear();
          setStatus("signedOut");
        } else {
          setStatus("failed");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [bootAttempt]);

  // A rejected refresh token anywhere in the app ends the session.
  useEffect(
    () =>
      session.subscribe((signedIn) => {
        if (!signedIn) {
          setUser(null);
          setStatus("signedOut");
          queryClient.clear();
        }
      }),
    [queryClient]
  );

  const complete = useCallback(async (res: AuthResponse) => {
    await session.set({ accessToken: res.accessToken, refreshToken: res.refreshToken });
    setUser(res.user);
    setStatus("signedIn");
  }, []);

  const signIn = useCallback(
    async (email: string, password: string) => {
      queryClient.clear();
      await complete(await authApi.login(email.trim(), password));
    },
    [complete, queryClient]
  );

  const signUp = useCallback(
    async (email: string, password: string) => {
      queryClient.clear();
      await complete(await authApi.register(email.trim(), password));
    },
    [complete, queryClient]
  );

  const signOut = useCallback(async () => {
    const refreshToken = session.tokens?.refreshToken;
    if (refreshToken) {
      try {
        await authApi.logout(refreshToken);
      } catch {
        // the local session is dropped regardless
      }
    }
    await session.clear();
  }, []);

  const afterAccountDeleted = useCallback(async () => {
    await session.clear();
  }, []);

  const retryBoot = useCallback(() => {
    setStatus("booting");
    setBootAttempt((n) => n + 1);
  }, []);

  const value = useMemo(
    () => ({ status, user, signIn, signUp, signOut, afterAccountDeleted, retryBoot }),
    [status, user, signIn, signUp, signOut, afterAccountDeleted, retryBoot]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
