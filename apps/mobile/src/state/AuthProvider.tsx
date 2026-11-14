import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { onSessionExpired } from "../api/client";
import { authApi } from "../api/endpoints";
import { clearTokens, currentTokens, loadTokens, saveTokens } from "../api/session";
import type { AuthSession, AuthUser } from "../api/types";
import { realtime } from "../realtime/socket";
import { queryClient } from "./queryClient";

type AuthState = {
  status: "booting" | "signedOut" | "signedIn";
  user: AuthUser | null;
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
      } catch {
        if (active) await forgetSession();
      }
    })();
    return () => {
      active = false;
    };
  }, [forgetSession]);

  const value = useMemo(() => ({ status, user, signIn, signOut, forgetSession }), [status, user, signIn, signOut, forgetSession]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return ctx;
}
