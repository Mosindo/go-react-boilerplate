import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { authApi } from "../api/endpoints";
import { setSessionListener } from "../api/client";
import type { AuthResponse, AuthUser } from "../api/types";
import { clearTokens, loadTokens, saveTokens } from "../store/tokenStore";
import { queryClient } from "./queryClient";

type Status = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: Status;
  user: AuthUser | null;
  sessionNotice: string | null;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  forgetSession: () => Promise<void>;
  clearNotice: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [sessionNotice, setSessionNotice] = useState<string | null>(null);

  const forgetSession = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, []);

  useEffect(() => {
    setSessionListener({
      onTokens: () => undefined,
      onExpired: () => {
        setSessionNotice("Votre session a expiré. Reconnectez-vous.");
        void forgetSession();
      }
    });
    return () => setSessionListener(null);
  }, [forgetSession]);

  useEffect(() => {
    let active = true;
    (async () => {
      const tokens = await loadTokens();
      if (!tokens) {
        if (active) {
          setStatus("signedOut");
        }
        return;
      }
      try {
        const me = await authApi.me();
        if (active) {
          setUser(me);
          setStatus("signedIn");
        }
      } catch {
        if (active) {
          await forgetSession();
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [forgetSession]);

  const applySession = useCallback(async (session: AuthResponse) => {
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    queryClient.clear();
    setSessionNotice(null);
    setUser(session.user);
    setStatus("signedIn");
  }, []);

  const signIn = useCallback(
    async (email: string, password: string) => applySession(await authApi.login(email.trim(), password)),
    [applySession]
  );

  const signUp = useCallback(
    async (email: string, password: string) => applySession(await authApi.register(email.trim(), password)),
    [applySession]
  );

  const signOut = useCallback(async () => {
    const tokens = await loadTokens();
    if (tokens?.refreshToken) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // Best effort: local sign out always succeeds.
      }
    }
    await forgetSession();
  }, [forgetSession]);

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      user,
      sessionNotice,
      signIn,
      signUp,
      signOut,
      forgetSession,
      clearNotice: () => setSessionNotice(null)
    }),
    [status, user, sessionNotice, signIn, signUp, signOut, forgetSession]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
