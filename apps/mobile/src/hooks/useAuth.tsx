import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { QueryClient, QueryClientProvider, useQuery, useQueryClient } from "@tanstack/react-query";
import { logout as logoutRequest, me } from "../api/auth";
import { ApiError, setSessionHooks } from "../api/client";
import { getSelf } from "../api/dating";
import type { AuthSession, AuthUser, SelfState } from "../api/types";
import { finishOnboardingFlow } from "../state/onboardingFlow";
import { clearTokens, getTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

type Status = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: Status;
  accessToken: string | null;
  user: AuthUser | null;
  self: SelfState | undefined;
  selfLoading: boolean;
  selfError: Error | null;
  refreshSelf: () => Promise<void>;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Wipes local state after the server already deleted the account. */
  endSession: () => Promise<void>;
};

export const SELF_QUERY_KEY = ["self"] as const;

const AuthContext = createContext<AuthContextValue | null>(null);

function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: (count, error) => !(error instanceof ApiError && error.status >= 400 && error.status < 500) && count < 2,
        staleTime: 15_000,
        refetchOnWindowFocus: false
      },
      mutations: { retry: 0 }
    }
  });
}

function AuthProviderInner({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [tokens, setTokens] = useState<AuthTokens | null>(null);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [status, setStatus] = useState<Status>("booting");
  const refreshTokenRef = useRef<string | null>(null);
  refreshTokenRef.current = tokens?.refreshToken ?? null;

  const endSession = useCallback(async () => {
    finishOnboardingFlow();
    await clearTokens();
    setTokens(null);
    setUser(null);
    setStatus("signedOut");
    queryClient.clear();
  }, [queryClient]);

  // The API client refreshes tokens on its own; mirror its outcomes here.
  useEffect(() => {
    setSessionHooks({
      onTokens: (next) => setTokens(next),
      onSessionExpired: () => {
        void endSession();
      }
    });
  }, [endSession]);

  useEffect(() => {
    let active = true;
    (async () => {
      const stored = await getTokens();
      if (!active) {
        return;
      }
      if (!stored) {
        setStatus("signedOut");
        return;
      }
      setTokens(stored);
      try {
        const current = await me(); // refreshes the access token when needed
        if (active) {
          setUser(current);
          setStatus("signedIn");
        }
      } catch (error) {
        if (!active) {
          return;
        }
        if (error instanceof ApiError && error.status === 401) {
          await endSession();
        } else {
          // Offline at launch: keep the stored session and let screens retry.
          setStatus("signedIn");
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [endSession]);

  const selfQuery = useQuery<SelfState, Error>({
    queryKey: SELF_QUERY_KEY,
    queryFn: getSelf,
    enabled: status === "signedIn"
  });

  const signIn = useCallback(
    async (session: AuthSession) => {
      queryClient.clear();
      finishOnboardingFlow();
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      setTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      setUser(session.user);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const signOut = useCallback(async () => {
    const refreshToken = refreshTokenRef.current;
    try {
      if (refreshToken) {
        await logoutRequest(refreshToken);
      }
    } catch {
      // Best effort: local cleanup always wins.
    }
    await endSession();
  }, [endSession]);

  const refreshSelf = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: SELF_QUERY_KEY });
  }, [queryClient]);

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      accessToken: tokens?.accessToken ?? null,
      user,
      self: selfQuery.data,
      selfLoading: selfQuery.isLoading,
      selfError: selfQuery.error,
      refreshSelf,
      signIn,
      signOut,
      endSession
    }),
    [endSession, refreshSelf, selfQuery.data, selfQuery.error, selfQuery.isLoading, signIn, signOut, status, tokens?.accessToken, user]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [client] = useState(createQueryClient);
  return (
    <QueryClientProvider client={client}>
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
