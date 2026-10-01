import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError, setSessionListeners } from "../api/client";
import { authApi } from "../api/endpoints";
import type { Account, AuthSession } from "../api/types";
import { clearImageCache } from "../components/AuthImage";
import { clearTokens, loadTokens, saveTokens } from "../lib/tokenStore";

type Status = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: Status;
  account: Account | null;
  bootError: string | null;
  retryBoot: () => void;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Called after the account was deleted server-side: wipe local state without calling logout. */
  forgetSession: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 15_000,
        refetchOnWindowFocus: false,
        retry: (count, error) => !(error instanceof ApiError && error.status >= 400 && error.status < 500) && count < 2
      }
    }
  });
}

export function AuthProvider({ children, client }: { children: React.ReactNode; client?: QueryClient }) {
  const [queryClient] = useState(() => client ?? makeQueryClient());
  const [status, setStatus] = useState<Status>("booting");
  const [account, setAccount] = useState<Account | null>(null);
  const [bootError, setBootError] = useState<string | null>(null);
  const [bootNonce, setBootNonce] = useState(0);

  const wipe = useCallback(async () => {
    await clearTokens();
    clearImageCache();
    queryClient.clear();
    setAccount(null);
    setStatus("signedOut");
  }, [queryClient]);

  useEffect(() => {
    setSessionListeners({ onExpired: () => void wipe() });
  }, [wipe]);

  useEffect(() => {
    let alive = true;
    (async () => {
      setBootError(null);
      const tokens = await loadTokens();
      if (!tokens) {
        if (alive) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.me();
        if (!alive) return;
        setAccount(me);
        setStatus("signedIn");
      } catch (e) {
        if (!alive) return;
        if (e instanceof ApiError && e.isNetwork) {
          setBootError(e.message);
        } else {
          await wipe();
        }
      }
    })();
    return () => {
      alive = false;
    };
  }, [bootNonce, wipe]);

  const apply = useCallback(
    async (session: AuthSession) => {
      queryClient.clear();
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      setAccount(session.user);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      account,
      bootError,
      retryBoot: () => setBootNonce((n) => n + 1),
      signIn: async (email, password) => apply(await authApi.login(email, password)),
      signUp: async (email, password) => apply(await authApi.register(email, password)),
      signOut: async () => {
        const tokens = await loadTokens();
        if (tokens) await authApi.logout(tokens.refreshToken).catch(() => undefined);
        await wipe();
      },
      forgetSession: wipe
    }),
    [status, account, bootError, apply, wipe]
  );

  return (
    <QueryClientProvider client={queryClient}>
      <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
    </QueryClientProvider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside <AuthProvider>");
  return ctx;
}
