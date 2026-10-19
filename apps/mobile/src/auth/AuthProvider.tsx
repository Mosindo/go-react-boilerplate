import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { login as loginRequest, logout as logoutRequest, me, register as registerRequest } from "../api/auth";
import { getTokens, setTokens, subscribeTokens } from "../api/client";
import type { AuthSession, AuthUser } from "../api/types";
import { loadTokens, persistTokens } from "./tokenStore";

type Status = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: Status;
  user: AuthUser | null;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state after the server deleted the account. */
  onAccountDeleted: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  // Persist tokens whenever the API layer rotates or drops them.
  useEffect(() => {
    return subscribeTokens((tokens) => {
      void persistTokens(tokens);
      if (!tokens) {
        setUser(null);
        setStatus("signedOut");
        queryClient.clear();
      }
    });
  }, [queryClient]);

  // Restore a previous session on launch.
  useEffect(() => {
    let active = true;
    void (async () => {
      const stored = await loadTokens();
      if (!active) return;
      if (!stored) {
        setStatus("signedOut");
        return;
      }
      setTokens(stored);
      try {
        const current = await me();
        if (!active) return;
        setUser(current);
        setStatus("signedIn");
      } catch {
        if (!active) return;
        // 401 already cleared the session; for network errors fall back to sign-in.
        if (getTokens()) setTokens(null);
        setStatus("signedOut");
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  const applySession = useCallback((session: AuthSession) => {
    setTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    setUser(session.user);
    setStatus("signedIn");
  }, []);

  const signIn = useCallback(
    async (email: string, password: string) => applySession(await loginRequest(email.trim(), password)),
    [applySession]
  );
  const signUp = useCallback(
    async (email: string, password: string) => applySession(await registerRequest(email.trim(), password)),
    [applySession]
  );

  const signOut = useCallback(async () => {
    const tokens = getTokens();
    if (tokens) {
      try {
        await logoutRequest(tokens.refreshToken);
      } catch {
        // best effort: local sign-out below always wins
      }
    }
    setTokens(null);
  }, []);

  const onAccountDeleted = useCallback(() => setTokens(null), []);

  const value = useMemo<AuthContextValue>(
    () => ({ status, user, signIn, signUp, signOut, onAccountDeleted }),
    [status, user, signIn, signUp, signOut, onAccountDeleted]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
