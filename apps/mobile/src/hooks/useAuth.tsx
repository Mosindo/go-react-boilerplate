import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import {
  QueryClient,
  QueryClientProvider,
  useMutation,
  useQueryClient,
  type UseMutationResult
} from "@tanstack/react-query";
import {
  login as loginRequest,
  logout as logoutRequest,
  me,
  register as registerRequest,
  type AuthSession,
  type AuthUser
} from "../api/auth";
import { ApiError, onSessionExpired } from "../api/client";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";

type AuthStatus = "booting" | "signedOut" | "signedIn";

type Credentials = { email: string; password: string };

type AuthContextValue = {
  status: AuthStatus;
  user: AuthUser | null;
  /** Message shown on the sign-in screen after a forced sign-out. */
  notice: string | null;
  applySession: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Used after account deletion: the tokens are already worthless. */
  forgetSession: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

function shouldRetry(failureCount: number, error: unknown): boolean {
  if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
    return false;
  }
  return failureCount < 2;
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: shouldRetry, staleTime: 15_000, refetchOnWindowFocus: false },
    mutations: { retry: 0 }
  }
});

function AuthProviderInner({ children }: { children: React.ReactNode }) {
  const client = useQueryClient();
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const forgetSession = useCallback(async () => {
    await clearTokens();
    client.clear();
    setUser(null);
    setStatus("signedOut");
  }, [client]);

  const applySession = useCallback(
    async (session: AuthSession) => {
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      client.clear();
      setNotice(null);
      setUser(session.user);
      setStatus("signedIn");
    },
    [client]
  );

  const signOut = useCallback(async () => {
    const tokens = await getTokens();
    if (tokens?.refreshToken) {
      try {
        await logoutRequest(tokens.refreshToken);
      } catch {
        // Local sign-out wins even when the server cannot be reached.
      }
    }
    await forgetSession();
  }, [forgetSession]);

  useEffect(() => {
    let active = true;
    (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const current = await me();
        if (active) {
          setUser(current);
          setStatus("signedIn");
        }
      } catch (error) {
        // Only a rejected session signs the user out; a network error keeps the tokens.
        if (active && error instanceof ApiError && error.status === 401) {
          await clearTokens();
          setStatus("signedOut");
        } else if (active) {
          setStatus("signedOut");
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
        setNotice("Votre session a expiré. Reconnectez-vous.");
        client.clear();
        setUser(null);
        setStatus("signedOut");
      }),
    [client]
  );

  const value = useMemo<AuthContextValue>(
    () => ({ status, user, notice, applySession, signOut, forgetSession }),
    [status, user, notice, applySession, signOut, forgetSession]
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

/** The signed-in user; only call below the auth gate. */
export function useCurrentUser(): AuthUser {
  const { user } = useAuth();
  if (!user) {
    throw new Error("useCurrentUser requires a signed-in user");
  }
  return user;
}

export function useLogin(): UseMutationResult<AuthSession, Error, Credentials> {
  const { applySession } = useAuth();
  return useMutation<AuthSession, Error, Credentials>({
    mutationFn: ({ email, password }) => loginRequest(email.trim(), password),
    onSuccess: applySession
  });
}

export function useRegister(): UseMutationResult<AuthSession, Error, Credentials> {
  const { applySession } = useAuth();
  return useMutation<AuthSession, Error, Credentials>({
    mutationFn: ({ email, password }) => registerRequest(email.trim(), password),
    onSuccess: applySession
  });
}
