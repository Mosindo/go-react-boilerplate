import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { ApiError, onSessionChange } from "../api/client";
import { authApi } from "../api/endpoints";
import { clearTokens, getTokens, saveTokens } from "../api/tokenStore";
import type { AuthSession, AuthUser } from "../api/types";
import { realtime } from "../api/realtime";
import { keys, queryClient } from "./queryClient";

type Status = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: Status;
  user: AuthUser | null;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state after the account was deleted server-side. */
  forgetSession: () => Promise<void>;
  refreshUser: () => Promise<AuthUser | null>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  const resetLocal = useCallback(async () => {
    realtime.stop();
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, []);

  const refreshUser = useCallback(async () => {
    try {
      const me = await authApi.me();
      setUser(me);
      queryClient.setQueryData(keys.me, me);
      return me;
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        await resetLocal();
      }
      return null;
    }
  }, [resetLocal]);

  useEffect(() => {
    onSessionChange((session) => {
      if (!session) {
        void resetLocal();
      } else {
        setUser(session.user);
      }
    });
    void (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        setStatus("signedOut");
        return;
      }
      const me = await refreshUser();
      setStatus(me ? "signedIn" : "signedOut");
    })();
    return () => onSessionChange(null);
  }, [refreshUser, resetLocal]);

  const signIn = useCallback(async (session: AuthSession) => {
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    queryClient.clear();
    setUser(session.user);
    setStatus("signedIn");
  }, []);

  const signOut = useCallback(async () => {
    const tokens = await getTokens();
    if (tokens?.refreshToken) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // Local sign-out still happens; the server session expires on its own.
      }
    }
    await resetLocal();
  }, [resetLocal]);

  const value = useMemo(
    () => ({ status, user, signIn, signOut, forgetSession: resetLocal, refreshUser }),
    [status, user, signIn, signOut, resetLocal, refreshUser]
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
