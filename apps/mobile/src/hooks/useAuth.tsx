import React, { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getTokens, onSessionChange, restoreSession, setSession } from "../api/client";
import { authApi } from "../api/endpoints";
import type { AuthResponse, AuthUser } from "../api/types";

type Status = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: Status;
  user: AuthUser | null;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  useEffect(() => {
    const unsubscribe = onSessionChange((tokens) => {
      if (!tokens) {
        setUser(null);
        setStatus("signedOut");
        queryClient.clear();
      }
    });
    (async () => {
      const tokens = await restoreSession();
      if (!tokens) {
        setStatus("signedOut");
        return;
      }
      try {
        setUser(await authApi.me());
        setStatus("signedIn");
      } catch {
        // Offline at boot with a stored session: stay signed in only if the
        // tokens survived (an invalid session is cleared by the client).
        setStatus(getTokens() ? "signedIn" : "signedOut");
      }
    })();
    return unsubscribe;
  }, [queryClient]);

  const apply = useCallback(
    async (res: AuthResponse) => {
      queryClient.clear();
      await setSession({ accessToken: res.accessToken, refreshToken: res.refreshToken });
      setUser(res.user);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const signIn = useCallback(async (email: string, password: string) => apply(await authApi.login(email.trim(), password)), [apply]);
  const signUp = useCallback(async (email: string, password: string) => apply(await authApi.register(email.trim(), password)), [apply]);

  const signOut = useCallback(async () => {
    const refresh = getTokens()?.refreshToken;
    if (refresh) {
      try {
        await authApi.logout(refresh);
      } catch {
        /* best effort: local sign-out always wins */
      }
    }
    await setSession(null);
  }, []);

  const deleteAccount = useCallback(async (password: string) => {
    await authApi.deleteAccount(password);
    await setSession(null);
  }, []);

  const value = useMemo(() => ({ status, user, signIn, signUp, signOut, deleteAccount }), [status, user, signIn, signUp, signOut, deleteAccount]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
