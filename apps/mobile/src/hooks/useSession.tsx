import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { authApi } from "../api/auth";
import { onSessionExpired } from "../api/client";
import type { AuthResponse, AuthUser } from "../api/types";
import { realtime } from "../api/realtime";
import { clearTokens, currentTokens, loadTokens, saveTokens } from "../store/tokenStore";
import { showToast } from "../ui/Toast";
import { queryClient } from "./queryClient";

type SessionStatus = "booting" | "signedOut" | "signedIn";

type SessionContextValue = {
  status: SessionStatus;
  user: AuthUser | null;
  signIn: (response: AuthResponse) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears the local session without calling the API (after account deletion). */
  forgetSession: () => Promise<void>;
};

const SessionContext = createContext<SessionContextValue | null>(null);

function SessionProviderInner({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<SessionStatus>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  const resetLocal = useCallback(async () => {
    realtime.stop();
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
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
        if (!active) return;
        setUser(me);
        setStatus("signedIn");
      } catch {
        if (!active) return;
        // The refresh flow already ran inside api(); reaching here means the
        // stored session is unusable (expired or the API is unreachable).
        setStatus("signedOut");
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  useEffect(
    () =>
      onSessionExpired(() => {
        void resetLocal();
        showToast("Votre session a expiré. Reconnectez-vous.", "info");
      }),
    [resetLocal]
  );

  const signIn = useCallback(async (response: AuthResponse) => {
    await saveTokens({ accessToken: response.accessToken, refreshToken: response.refreshToken });
    queryClient.clear();
    setUser(response.user);
    setStatus("signedIn");
  }, []);

  const signOut = useCallback(async () => {
    const tokens = currentTokens();
    try {
      if (tokens) {
        await authApi.logout(tokens.refreshToken);
      }
    } catch {
      // Local sign-out still happens.
    }
    await resetLocal();
  }, [resetLocal]);

  const value = useMemo(() => ({ status, user, signIn, signOut, forgetSession: resetLocal }), [status, user, signIn, signOut, resetLocal]);
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
    throw new Error("useSession must be used inside SessionProvider");
  }
  return ctx;
}
