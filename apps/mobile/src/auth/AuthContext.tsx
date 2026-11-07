import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "../api/services";
import { getTokens, request, setSessionListener, setTokens } from "../api/http";
import type { AuthResponse, AuthUser } from "../api/types";

type Status = "booting" | "signedOut" | "signedIn";

type AuthValue = {
  status: Status;
  user: AuthUser | null;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
};

const AuthContext = createContext<AuthValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const queryClient = useQueryClient();

  const clearSession = useCallback(async () => {
    await setTokens(null);
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, [queryClient]);

  useEffect(() => {
    setSessionListener({
      onTokens: () => undefined,
      onSignedOut: () => {
        queryClient.clear();
        setUser(null);
        setStatus("signedOut");
      },
    });
    return () => setSessionListener(null);
  }, [queryClient]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        if (!cancelled) setStatus("signedOut");
        return;
      }
      try {
        const me = await request<AuthUser>("/me");
        if (!cancelled) {
          setUser(me);
          setStatus("signedIn");
        }
      } catch {
        // Offline at startup keeps tokens; a rejected session is cleared by the refresh logic.
        if (!cancelled) setStatus((await getTokens()) ? "signedIn" : "signedOut");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const accept = useCallback(
    async (res: AuthResponse) => {
      queryClient.clear();
      await setTokens({ accessToken: res.accessToken, refreshToken: res.refreshToken });
      setUser(res.user);
      setStatus("signedIn");
    },
    [queryClient],
  );

  const value = useMemo<AuthValue>(
    () => ({
      status,
      user,
      signIn: async (email, password) => accept(await authApi.login(email.trim(), password)),
      signUp: async (email, password) => accept(await authApi.register(email.trim(), password)),
      signOut: async () => {
        const tokens = await getTokens();
        if (tokens) await authApi.logout(tokens.refreshToken).catch(() => undefined);
        await clearSession();
      },
      deleteAccount: async (password) => {
        await authApi.deleteAccount(password);
        await clearSession();
      },
    }),
    [status, user, accept, clearSession],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
