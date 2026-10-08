import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { QueryClientProvider, useQueryClient } from "@tanstack/react-query";
import { logout as logoutRequest, type AuthSession } from "../api/auth";
import { queryClient } from "../api/queryClient";
import { onSessionExpired } from "../api/sessionEvents";
import { clearTokens, getRefreshToken, getTokens, saveTokens } from "../store/tokenStore";

export type AuthStatus = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: AuthStatus;
  /** Set when the session ended without the user asking for it. */
  sessionExpired: boolean;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state after the server already removed the account. */
  forgetSession: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

function AuthProviderInner({ children }: { children: React.ReactNode }) {
  const client = useQueryClient();
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [sessionExpired, setSessionExpired] = useState(false);

  const reset = useCallback(async () => {
    await clearTokens();
    client.clear();
    setStatus("signedOut");
  }, [client]);

  useEffect(() => {
    let active = true;
    void getTokens().then((tokens) => {
      if (active) setStatus(tokens ? "signedIn" : "signedOut");
    });
    return () => {
      active = false;
    };
  }, []);

  useEffect(
    () =>
      onSessionExpired(() => {
        setSessionExpired(true);
        void reset();
      }),
    [reset]
  );

  const signIn = useCallback(
    async (session: AuthSession) => {
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      client.clear();
      setSessionExpired(false);
      setStatus("signedIn");
    },
    [client]
  );

  const signOut = useCallback(async () => {
    const refreshToken = getRefreshToken();
    try {
      if (refreshToken) await logoutRequest(refreshToken);
    } catch {
      // Best effort: local sign-out always wins.
    }
    setSessionExpired(false);
    await reset();
  }, [reset]);

  const value = useMemo<AuthContextValue>(
    () => ({ status, sessionExpired, signIn, signOut, forgetSession: reset }),
    [status, sessionExpired, signIn, signOut, reset]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProviderInner>{children}</AuthProviderInner>
    </QueryClientProvider>
  );
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within AuthProvider");
  return context;
}
