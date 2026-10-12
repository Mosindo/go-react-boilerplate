import { QueryClient, QueryClientProvider, useQueryClient } from "@tanstack/react-query";
import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { login as loginRequest, logout as logoutRequest, me, register as registerRequest, type AuthSession } from "../api/auth";
import { setSessionLostHandler } from "../api/client";
import type { AuthUser } from "../api/types";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";

type AuthState = {
  user: AuthUser | null;
  isBooting: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state after the account has been deleted server-side. */
  forgetSession: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000, refetchOnWindowFocus: false },
    mutations: { retry: 0 }
  }
});

function Inner({ children }: { children: React.ReactNode }) {
  const client = useQueryClient();
  const [user, setUser] = useState<AuthUser | null>(null);
  const [isBooting, setBooting] = useState(true);

  const forgetSession = useCallback(async () => {
    await clearTokens();
    client.clear();
    setUser(null);
  }, [client]);

  useEffect(() => {
    setSessionLostHandler(() => {
      void forgetSession();
    });
    return () => setSessionLostHandler(null);
  }, [forgetSession]);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        if (await getTokens()) {
          const current = await me();
          if (active) {
            setUser(current);
          }
        }
      } catch {
        // An invalid session is cleared by the API client; otherwise stay signed out.
      } finally {
        if (active) {
          setBooting(false);
        }
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  const apply = useCallback(
    async (session: AuthSession) => {
      client.clear();
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      setUser(session.user);
    },
    [client]
  );

  const value = useMemo<AuthState>(
    () => ({
      user,
      isBooting,
      signIn: async (email, password) => apply(await loginRequest(email.trim(), password)),
      signUp: async (email, password) => apply(await registerRequest(email.trim(), password)),
      signOut: async () => {
        const tokens = await getTokens();
        try {
          if (tokens) {
            await logoutRequest(tokens.refreshToken);
          }
        } catch {
          // best effort: local cleanup always happens
        }
        await forgetSession();
      },
      forgetSession
    }),
    [apply, forgetSession, isBooting, user]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <Inner>{children}</Inner>
    </QueryClientProvider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
