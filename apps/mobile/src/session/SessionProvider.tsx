import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { authApi } from "../api/auth";
import { ApiError, onSessionChange } from "../api/client";
import type { AuthSession, AuthUser } from "../api/types";
import { queryClient } from "../lib/queryClient";
import { clearTokens, currentTokens, loadTokens, saveTokens } from "./tokenStore";

type SessionState = {
  status: "booting" | "signedOut" | "signedIn";
  user: AuthUser | null;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: (options?: { remote?: boolean }) => Promise<void>;
};

const SessionContext = createContext<SessionState | null>(null);

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<SessionState["status"]>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const tokens = await loadTokens();
      if (!tokens) {
        if (!cancelled) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.me();
        if (!cancelled) {
          setUser(me);
          setStatus("signedIn");
        }
      } catch (error) {
        // Offline at boot keeps the stored session; an invalid one is dropped.
        if (error instanceof ApiError && error.status === 401) await clearTokens();
        if (!cancelled) setStatus("signedOut");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(
    () =>
      onSessionChange((session) => {
        if (session) {
          setUser(session.user);
        } else {
          queryClient.clear();
          setUser(null);
          setStatus("signedOut");
        }
      }),
    []
  );

  const signIn = useCallback(async (session: AuthSession) => {
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    queryClient.clear();
    setUser(session.user);
    setStatus("signedIn");
  }, []);

  const signOut = useCallback(async (options?: { remote?: boolean }) => {
    const tokens = currentTokens();
    if (options?.remote !== false && tokens?.refreshToken) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // Local sign-out must succeed even offline.
      }
    }
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, []);

  const value = useMemo(() => ({ status, user, signIn, signOut }), [status, user, signIn, signOut]);
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionState {
  const ctx = useContext(SessionContext);
  if (!ctx) throw new Error("useSession must be used inside SessionProvider");
  return ctx;
}
