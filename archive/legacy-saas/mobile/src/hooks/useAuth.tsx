import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import {
  QueryClient,
  QueryClientProvider,
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult
} from "@tanstack/react-query";
import {
  login as loginRequest,
  logout as logoutRequest,
  me,
  refreshSession as refreshSessionRequest,
  register as registerRequest,
  type AuthSession,
  type AuthUser
} from "../api/auth";
import {
  beginGlobalLoading,
  clearGlobalError,
  endGlobalLoading
} from "../shared/feedback";
import { clearTokens, getTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

type AuthCredentials = {
  email: string;
  password: string;
};

type AuthContextValue = {
  accessToken: string | null;
  refreshToken: string | null;
  user: AuthUser | null;
  isBooting: boolean;
  isAuthenticated: boolean;
  isLoggingOut: boolean;
  authError: string | null;
  currentUserQuery: UseQueryResult<AuthUser, Error>;
  applySession: (session: AuthSession) => Promise<void>;
  clearAuthError: () => void;
  logout: () => Promise<void>;
};

const CURRENT_USER_QUERY_KEY = ["auth", "current-user"] as const;

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000,
      refetchOnWindowFocus: false
    },
    mutations: {
      retry: 0
    }
  }
});

const AuthContext = createContext<AuthContextValue | null>(null);

function toStoredTokens(session: AuthSession): AuthTokens {
  return {
    accessToken: session.accessToken,
    refreshToken: session.refreshToken
  };
}

function formatAuthError(error: unknown, fallback: string): string {
  if (!(error instanceof Error) || !error.message) {
    return fallback;
  }
  return error.message;
}

function AuthProviderInner({ children }: { children: React.ReactNode }) {
  const queryClientInstance = useQueryClient();
  const [tokens, setTokens] = useState<AuthTokens | null>(null);
  const [bootstrapping, setBootstrapping] = useState(true);
  const [authError, setAuthError] = useState<string | null>(null);
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [refreshAttemptedFor, setRefreshAttemptedFor] = useState<string | null>(null);

  const currentUserQuery = useQuery<AuthUser, Error>({
    queryKey: [...CURRENT_USER_QUERY_KEY, tokens?.accessToken ?? "guest"],
    enabled: !bootstrapping && Boolean(tokens?.accessToken),
    queryFn: () => me(),
    retry: false
  });

  const clearSession = useCallback(async () => {
    await clearTokens();
    setTokens(null);
    setAuthError(null);
    setRefreshAttemptedFor(null);
    queryClientInstance.removeQueries({ queryKey: CURRENT_USER_QUERY_KEY });
  }, [queryClientInstance]);

  const applySession = useCallback(
    async (session: AuthSession) => {
      await saveTokens(toStoredTokens(session));
      setTokens(toStoredTokens(session));
      setAuthError(null);
      clearGlobalError();
      setRefreshAttemptedFor(null);
      queryClientInstance.setQueryData([...CURRENT_USER_QUERY_KEY, session.accessToken], session.user);
    },
    [queryClientInstance]
  );

  const clearAuthError = useCallback(() => {
    setAuthError(null);
  }, []);

  const refreshCurrentSession = useCallback(async () => {
    if (!tokens?.refreshToken) {
      await clearSession();
      return false;
    }

    beginGlobalLoading("Refreshing session...");
    try {
      const nextSession = await refreshSessionRequest(tokens.refreshToken);
      await applySession(nextSession);
      return true;
    } catch (error) {
      await clearSession();
      setAuthError(formatAuthError(error, "Your session expired. Please login again."));
      return false;
    } finally {
      endGlobalLoading();
    }
  }, [applySession, clearSession, tokens?.refreshToken]);

  const logout = useCallback(async () => {
    setIsLoggingOut(true);
    beginGlobalLoading("Signing out...");
    try {
      if (tokens?.refreshToken) {
        await logoutRequest(tokens.refreshToken);
      }
    } catch {
      // Best-effort logout. Local cleanup still wins.
    } finally {
      await clearSession();
      setIsLoggingOut(false);
      endGlobalLoading();
    }
  }, [clearSession, tokens?.refreshToken]);

  useEffect(() => {
    let active = true;

    async function bootstrap() {
      try {
        const restored = await getTokens();
        if (!active) {
          return;
        }
        setTokens(restored);
      } catch {
        if (active) {
          setTokens(null);
        }
      } finally {
        if (active) {
          setBootstrapping(false);
        }
      }
    }

    bootstrap();

    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (bootstrapping || !tokens?.accessToken || !currentUserQuery.isError) {
      return;
    }

    if (tokens.refreshToken && refreshAttemptedFor !== tokens.refreshToken) {
      setRefreshAttemptedFor(tokens.refreshToken);
      void refreshCurrentSession();
      return;
    }

    void clearSession();
    setAuthError("Your session expired. Please login again.");
  }, [
    bootstrapping,
    clearSession,
    currentUserQuery.isError,
    refreshAttemptedFor,
    refreshCurrentSession,
    tokens?.accessToken,
    tokens?.refreshToken
  ]);

  const value = useMemo<AuthContextValue>(
    () => ({
      accessToken: tokens?.accessToken ?? null,
      refreshToken: tokens?.refreshToken ?? null,
      user: currentUserQuery.data ?? null,
      isBooting: bootstrapping || (!!tokens?.accessToken && currentUserQuery.isLoading),
      isAuthenticated: Boolean(tokens?.accessToken && currentUserQuery.data),
      isLoggingOut,
      authError,
      currentUserQuery,
      applySession,
      clearAuthError,
      logout
    }),
    [applySession, authError, bootstrapping, clearAuthError, currentUserQuery, isLoggingOut, logout, tokens]
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

export function useCurrentUser(): UseQueryResult<AuthUser, Error> {
  return useAuth().currentUserQuery;
}

export function useLogin(): UseMutationResult<AuthSession, Error, AuthCredentials> {
  const { applySession, clearAuthError } = useAuth();

  return useMutation<AuthSession, Error, AuthCredentials>({
    mutationFn: async ({ email, password }) => {
      clearAuthError();
      beginGlobalLoading("Signing in...");
      try {
        return await loginRequest(email, password);
      } finally {
        endGlobalLoading();
      }
    },
    onSuccess: async (session) => {
      await applySession(session);
    }
  });
}

export function useRegister(): UseMutationResult<AuthSession, Error, AuthCredentials> {
  const { applySession, clearAuthError } = useAuth();

  return useMutation<AuthSession, Error, AuthCredentials>({
    mutationFn: async ({ email, password }) => {
      clearAuthError();
      beginGlobalLoading("Creating account...");
      try {
        return await registerRequest(email, password);
      } finally {
        endGlobalLoading();
      }
    },
    onSuccess: async (session) => {
      await applySession(session);
    }
  });
}
