import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "../api/endpoints";
import { ApiError, onSessionExpired } from "../api/client";
import { clearTokens, currentTokens, loadTokens, saveTokens } from "../api/tokenStore";
import type { AuthSession, AuthUser } from "../api/types";

type Status = "booting" | "signedOut" | "signedIn" | "unreachable";

type SessionValue = {
  status: Status;
  user: AuthUser | null;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  retryBoot: () => void;
};

const SessionContext = createContext<SessionValue | null>(null);

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [bootAttempt, setBootAttempt] = useState(0);

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
      } catch (err) {
        if (!active) return;
        // 401 means the refresh token is dead as well; anything else is a connectivity problem.
        if (err instanceof ApiError && err.status !== 0 && err.status < 500) {
          await clearTokens();
          setStatus("signedOut");
        } else {
          setStatus("unreachable");
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [bootAttempt]);

  const reset = useCallback(() => {
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, [queryClient]);

  useEffect(() => onSessionExpired(reset), [reset]);

  const signIn = useCallback(
    async (session: AuthSession) => {
      queryClient.clear();
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      setUser(session.user);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const signOut = useCallback(async () => {
    const refreshToken = currentTokens()?.refreshToken;
    if (refreshToken) {
      try {
        await authApi.logout(refreshToken);
      } catch {
        // Best effort: local sign-out always wins.
      }
    }
    await clearTokens();
    reset();
  }, [reset]);

  const retryBoot = useCallback(() => {
    setStatus("booting");
    setBootAttempt((n) => n + 1);
  }, []);

  const value = useMemo(
    () => ({ status, user, signIn, signOut, retryBoot }),
    [status, user, signIn, signOut, retryBoot]
  );
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionValue {
  const ctx = useContext(SessionContext);
  if (!ctx) throw new Error("useSession must be used inside SessionProvider");
  return ctx;
}
