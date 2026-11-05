import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError, api, setSessionExpiredHandler } from "../api/client";
import { authApi } from "../api/endpoints";
import type { Account } from "../api/types";
import { getTokens, loadTokens, saveTokens, subscribeTokens } from "../store/tokenStore";

type Status = "booting" | "anonymous" | "authenticated" | "unreachable";

type AuthValue = {
  status: Status;
  user: Account | null;
  accessToken: string | null;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
  retryBoot: () => void;
};

const AuthContext = createContext<AuthValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<Account | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [bootNonce, setBootNonce] = useState(0);

  const clearSession = useCallback(() => {
    setUser(null);
    setStatus("anonymous");
    queryClient.clear();
  }, [queryClient]);

  useEffect(() => subscribeTokens((t) => setAccessToken(t?.accessToken ?? null)), []);

  useEffect(() => {
    setSessionExpiredHandler(clearSession);
    return () => setSessionExpiredHandler(null);
  }, [clearSession]);

  useEffect(() => {
    let cancelled = false;
    setStatus("booting");
    (async () => {
      const tokens = await loadTokens();
      if (cancelled) return;
      setAccessToken(tokens?.accessToken ?? null);
      if (!tokens) {
        setStatus("anonymous");
        return;
      }
      try {
        const me = await api<Account>("/me");
        if (cancelled) return;
        setUser(me);
        setAccessToken(getTokens()?.accessToken ?? null);
        setStatus("authenticated");
      } catch (err) {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 0) {
          setStatus("unreachable");
        } else {
          await saveTokens(null);
          setStatus("anonymous");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [bootNonce]);

  const startSession = useCallback(
    async (res: { accessToken: string; refreshToken: string; user: Account }) => {
      queryClient.clear();
      await saveTokens({ accessToken: res.accessToken, refreshToken: res.refreshToken });
      setUser(res.user);
      setStatus("authenticated");
    },
    [queryClient]
  );

  const signIn = useCallback(
    async (email: string, password: string) => startSession(await authApi.login(email, password)),
    [startSession]
  );
  const signUp = useCallback(
    async (email: string, password: string) =>
      startSession(await authApi.register(email, password)),
    [startSession]
  );

  const signOut = useCallback(async () => {
    const tokens = getTokens();
    if (tokens) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        /* best effort: the local session is dropped regardless */
      }
    }
    await saveTokens(null);
    clearSession();
  }, [clearSession]);

  const deleteAccount = useCallback(
    async (password: string) => {
      await authApi.deleteAccount(password);
      await saveTokens(null);
      clearSession();
    },
    [clearSession]
  );

  const value = useMemo<AuthValue>(
    () => ({
      status,
      user,
      accessToken,
      signIn,
      signUp,
      signOut,
      deleteAccount,
      retryBoot: () => setBootNonce((n) => n + 1)
    }),
    [status, user, accessToken, signIn, signUp, signOut, deleteAccount]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
