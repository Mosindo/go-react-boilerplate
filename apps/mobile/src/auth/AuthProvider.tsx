import React, { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { authApi } from "../api/endpoints";
import type { Account, AuthResponse } from "../api/types";
import { queryClient } from "../queryClient";
import { clearTokens, getTokens, onTokensChange, restoreTokens, setTokens } from "./session";

type Status = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: Status;
  account: Account | null;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<Status>("booting");
  const [account, setAccount] = useState<Account | null>(null);

  useEffect(() => {
    let active = true;
    (async () => {
      const tokens = await restoreTokens();
      if (!tokens) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.me();
        if (active) {
          setAccount(me);
          setStatus("signedIn");
        }
      } catch {
        // The client already cleared tokens if the session is no longer valid.
        if (active) setStatus(getTokens() ? "signedIn" : "signedOut");
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  // Any place that clears the tokens (failed refresh, revoked session) signs out.
  useEffect(
    () =>
      onTokensChange((tokens) => {
        if (!tokens) {
          setAccount(null);
          setStatus("signedOut");
          queryClient.clear();
        }
      }),
    []
  );

  const applyAuth = useCallback(async (response: AuthResponse) => {
    queryClient.clear();
    await setTokens({ accessToken: response.accessToken, refreshToken: response.refreshToken });
    setAccount(response.user);
    setStatus("signedIn");
  }, []);

  const signIn = useCallback(async (email: string, password: string) => {
    await applyAuth(await authApi.login(email.trim(), password));
  }, [applyAuth]);

  const signUp = useCallback(async (email: string, password: string) => {
    await applyAuth(await authApi.register(email.trim(), password));
  }, [applyAuth]);

  const signOut = useCallback(async () => {
    const tokens = getTokens();
    if (tokens) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // Local sign-out always wins.
      }
    }
    await clearTokens();
  }, []);

  const deleteAccount = useCallback(async (password: string) => {
    await authApi.deleteAccount(password);
    await clearTokens();
  }, []);

  const value = useMemo(
    () => ({ status, account, signIn, signUp, signOut, deleteAccount }),
    [status, account, signIn, signUp, signOut, deleteAccount]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
