import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { ApiError, onSessionExpired } from "../api/client";
import { authApi } from "../api/endpoints";
import { clearTokens, currentTokens, loadTokens, saveTokens } from "../api/session";
import type { AuthSession, AuthUser } from "../api/types";
import { realtime } from "../realtime/socket";
import { queryClient } from "./queryClient";

type AuthState = {
  /** "unreachable": tokens exist but the API could not be reached at start. */
  status: "booting" | "signedOut" | "signedIn" | "unreachable";
  user: AuthUser | null;
  retry: () => void;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Local cleanup only (account deleted, session revoked). */
  forgetSession: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthState["status"]>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  const forgetSession = useCallback(async () => {
    realtime.stop();
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, []);

  const signIn = useCallback(async (session: AuthSession) => {
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    queryClient.clear();
    setUser(session.user);
    setStatus("signedIn");
  }, []);

  const signOut = useCallback(async () => {
    const refreshToken = currentTokens()?.refreshToken;
    if (refreshToken) {
      await authApi.logout(refreshToken).catch(() => undefined);
    }
    await forgetSession();
  }, [forgetSession]);

  useEffect(() => onSessionExpired(() => void forgetSession()), [forgetSession]);

  const [attempt, setAttempt] = useState(0);
  const retry = useCallback(() => {
    setStatus("booting");
    setAttempt((n) => n + 1);
  }, []);

  useEffect(() => {
    let active = true;
    (async () => {
      const tokens = await loadTokens();
      if (!tokens) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.me();
        if (active) {
          setUser(me);
          setStatus("signedIn");
        }
      } catch (error) {
        if (!active) return;
        // Only a rejected session signs the user out; a network failure at
        // start-up must not wipe a valid session.
        if (error instanceof ApiError && error.status === 401) {
          await forgetSession();
        } else {
          setStatus("unreachable");
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [forgetSession, attempt]);

  const value = useMemo(
    () => ({ status, user, retry, signIn, signOut, forgetSession }),
    [status, user, retry, signIn, signOut, forgetSession]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return ctx;
}
