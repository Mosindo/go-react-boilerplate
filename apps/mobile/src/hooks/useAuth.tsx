import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { QueryClient, QueryClientProvider, useMutation, type UseMutationResult } from "@tanstack/react-query";
import {
  login as loginRequest,
  logout as logoutRequest,
  me,
  register as registerRequest,
  type AuthSession,
  type AuthUser
} from "../api/auth";
import { ApiError, onSessionExpired } from "../api/client";
import { clearDecisions } from "../lib/decisions";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000, refetchOnWindowFocus: false },
    mutations: { retry: 0 }
  }
});

type AuthStatus = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: AuthStatus;
  user: AuthUser | null;
  /** One-shot message for the sign-in screen (expired session, deleted account...). */
  notice: string | null;
  clearNotice: () => void;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: (notice?: string) => Promise<void>;
};

type Credentials = { email: string; password: string };

const AuthContext = createContext<AuthContextValue | null>(null);

function AuthProviderInner({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const reset = useCallback((message: string | null) => {
    queryClient.clear(); // never let one account's data leak into the next session
    clearDecisions();
    setUser(null);
    setNotice(message);
    setStatus("signedOut");
  }, []);

  const signIn = useCallback(async (session: AuthSession) => {
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    queryClient.clear();
    clearDecisions();
    setUser(session.user);
    setNotice(null);
    setStatus("signedIn");
  }, []);

  const signOut = useCallback(
    async (message?: string) => {
      const tokens = await getTokens();
      if (tokens?.refreshToken) {
        try {
          await logoutRequest(tokens.refreshToken);
        } catch {
          // Best effort: the local session is cleared either way.
        }
      }
      await clearTokens();
      reset(message ?? null);
    },
    [reset]
  );

  useEffect(() => {
    let active = true;
    (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        if (active) {
          setStatus("signedOut");
        }
        return;
      }
      try {
        const current = await me();
        if (active) {
          setUser(current);
          setStatus("signedIn");
        }
      } catch (error) {
        // An HTTP error means the session is dead (the client already cleared it).
        // Anything else is a network failure: stored tokens are kept for the next attempt.
        if (active) {
          setStatus("signedOut");
          if (!(error instanceof ApiError)) {
            setNotice("Connexion impossible pour le moment. Réessayez.");
          }
        }
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  useEffect(
    () =>
      onSessionExpired(() => {
        reset("Votre session a expiré. Reconnectez-vous.");
      }),
    [reset]
  );

  const value = useMemo<AuthContextValue>(
    () => ({ status, user, notice, clearNotice: () => setNotice(null), signIn, signOut }),
    [notice, signIn, signOut, status, user]
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
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}

/** For screens that only render when signed in. */
export function useSignedInUser(): AuthUser {
  const { user } = useAuth();
  if (!user) {
    throw new Error("useSignedInUser used while signed out");
  }
  return user;
}

export function useLogin(): UseMutationResult<AuthSession, Error, Credentials> {
  const { signIn } = useAuth();
  return useMutation<AuthSession, Error, Credentials>({
    mutationFn: ({ email, password }) => loginRequest(email, password),
    onSuccess: signIn
  });
}

export function useRegister(): UseMutationResult<AuthSession, Error, Credentials> {
  const { signIn } = useAuth();
  return useMutation<AuthSession, Error, Credentials>({
    mutationFn: ({ email, password }) => registerRequest(email, password),
    onSuccess: signIn
  });
}
