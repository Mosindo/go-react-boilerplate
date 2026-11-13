import { QueryClientProvider } from "@tanstack/react-query";
import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { authApi } from "../api/auth";
import { setSessionExpiredHandler } from "../api/client";
import { getTokens, restoreTokens, setTokens, subscribeTokens } from "../api/session";
import type { AuthSession } from "../api/types";
import { queryClient } from "./queryClient";

type Status = "booting" | "signedOut" | "signedIn";

type SessionContextValue = {
  status: Status;
  userId: string | null;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state after the server already ended the session. */
  forgetSession: () => Promise<void>;
};

const SessionContext = createContext<SessionContextValue | null>(null);

function SessionProviderInner({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<Status>("booting");
  const [userId, setUserId] = useState<string | null>(null);

  const forgetSession = useCallback(async () => {
    await setTokens(null);
    queryClient.clear();
    setUserId(null);
    setStatus("signedOut");
  }, []);

  useEffect(() => {
    setSessionExpiredHandler(() => {
      void forgetSession();
    });
    const unsubscribe = subscribeTokens((tokens) => {
      if (!tokens) {
        setStatus((s) => (s === "booting" ? s : "signedOut"));
      }
    });
    let active = true;
    (async () => {
      const tokens = await restoreTokens();
      if (!tokens) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.me();
        if (active) {
          setUserId(me.id);
          setStatus("signedIn");
        }
      } catch {
        if (active) {
          await setTokens(null);
          setStatus("signedOut");
        }
      }
    })();
    return () => {
      active = false;
      unsubscribe();
      setSessionExpiredHandler(null);
    };
  }, [forgetSession]);

  const signIn = useCallback(async (session: AuthSession) => {
    queryClient.clear();
    await setTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    setUserId(session.user.id);
    setStatus("signedIn");
  }, []);

  const signOut = useCallback(async () => {
    const refreshToken = getTokens()?.refreshToken;
    if (refreshToken) {
      try {
        await authApi.logout(refreshToken);
      } catch {
        // Best effort: local sign-out always succeeds.
      }
    }
    await forgetSession();
  }, [forgetSession]);

  const value = useMemo(
    () => ({ status, userId, signIn, signOut, forgetSession }),
    [forgetSession, signIn, signOut, status, userId]
  );
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function SessionProvider({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <SessionProviderInner>{children}</SessionProviderInner>
    </QueryClientProvider>
  );
}

export function useSession(): SessionContextValue {
  const ctx = useContext(SessionContext);
  if (!ctx) {
    throw new Error("useSession must be used within SessionProvider");
  }
  return ctx;
}
