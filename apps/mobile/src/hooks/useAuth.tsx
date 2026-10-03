import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { QueryClient, QueryClientProvider, useMutation, useQueryClient, type UseMutationResult } from "@tanstack/react-query";
import {
  login as loginRequest,
  logout as logoutRequest,
  me,
  refreshSession as refreshSessionRequest,
  register as registerRequest,
  type AuthSession,
  type AuthUser
} from "../api/auth";
import { setRefreshHandler } from "../api/session";
import { beginGlobalLoading, endGlobalLoading } from "../shared/feedback";
import { clearTokens, getTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

type AuthCredentials = { email: string; password: string };

type AuthContextValue = {
  accessToken: string | null;
  user: AuthUser | null;
  isBooting: boolean;
  isAuthenticated: boolean;
  authNotice: string | null;
  applySession: (session: AuthSession) => Promise<void>;
  clearAuthNotice: () => void;
  logout: () => Promise<void>;
  /** Drops local state after the server already removed the account. */
  endSession: () => Promise<void>;
};

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000, refetchOnWindowFocus: false },
    mutations: { retry: 0 }
  }
});

const AuthContext = createContext<AuthContextValue | null>(null);

function AuthProviderInner({ children }: { children: React.ReactNode }) {
  const client = useQueryClient();
  const [tokens, setTokens] = useState<AuthTokens | null>(null);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [booting, setBooting] = useState(true);
  const [authNotice, setAuthNotice] = useState<string | null>(null);
  const tokensRef = useRef<AuthTokens | null>(null);

  const endSession = useCallback(async () => {
    tokensRef.current = null;
    await clearTokens();
    setTokens(null);
    setUser(null);
    client.clear();
  }, [client]);

  const applySession = useCallback(async (session: AuthSession) => {
    const next = { accessToken: session.accessToken, refreshToken: session.refreshToken };
    tokensRef.current = next;
    await saveTokens(next);
    setTokens(next);
    setUser(session.user);
    setAuthNotice(null);
  }, []);

  // The HTTP client calls this on a 401: rotate the refresh token once, or sign the user out.
  useEffect(() => {
    setRefreshHandler(async () => {
      const current = tokensRef.current ?? (await getTokens());
      if (!current?.refreshToken) {
        return false;
      }
      try {
        const session = await refreshSessionRequest(current.refreshToken);
        await applySession(session);
        return true;
      } catch (error) {
        const status = (error as { status?: number }).status;
        // Only a definitive rejection ends the session; a network blip must not log people out.
        if (status === 401 || status === 400) {
          await endSession();
          setAuthNotice("Your session expired. Please sign in again.");
        }
        return false;
      }
    });
    return () => setRefreshHandler(null);
  }, [applySession, endSession]);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const restored = await getTokens();
        if (!restored || !active) {
          return;
        }
        tokensRef.current = restored;
        setTokens(restored);
        const current = await me();
        if (active) {
          setUser(current);
        }
      } catch (error) {
        const status = (error as { status?: number }).status;
        if (active && (status === 401 || status === 404)) {
          await endSession();
        }
      } finally {
        if (active) {
          setBooting(false);
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [endSession]);

  const logout = useCallback(async () => {
    beginGlobalLoading("Signing out...");
    try {
      const refreshToken = tokensRef.current?.refreshToken;
      if (refreshToken) {
        await logoutRequest(refreshToken);
      }
    } catch {
      // Best effort: local cleanup always wins.
    } finally {
      await endSession();
      endGlobalLoading();
    }
  }, [endSession]);

  const value = useMemo<AuthContextValue>(
    () => ({
      accessToken: tokens?.accessToken ?? null,
      user,
      isBooting: booting,
      isAuthenticated: Boolean(tokens?.accessToken && user),
      authNotice,
      applySession,
      clearAuthNotice: () => setAuthNotice(null),
      logout,
      endSession
    }),
    [applySession, authNotice, booting, endSession, logout, tokens, user]
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

export function useLogin(): UseMutationResult<AuthSession, Error, AuthCredentials> {
  const { applySession } = useAuth();
  return useMutation<AuthSession, Error, AuthCredentials>({
    mutationFn: ({ email, password }) => loginRequest(email.trim(), password),
    onSuccess: applySession
  });
}

export function useRegister(): UseMutationResult<AuthSession, Error, AuthCredentials> {
  const { applySession } = useAuth();
  return useMutation<AuthSession, Error, AuthCredentials>({
    mutationFn: ({ email, password }) => registerRequest(email.trim(), password),
    onSuccess: applySession
  });
}
