import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError, setSessionBridge } from "../api/client";
import { authApi } from "../api/endpoints";
import type { AuthSession, AuthUser } from "../api/types";
import { clearTokens, loadTokens, saveTokens, type StoredTokens } from "./tokenStore";

type AuthState = {
  status: "booting" | "signedOut" | "signedIn";
  user: AuthUser | null;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state only (used after account deletion or session loss). */
  forgetSession: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const tokensRef = useRef<StoredTokens | null>(null);
  const [status, setStatus] = useState<AuthState["status"]>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  const forgetSession = useCallback(async () => {
    tokensRef.current = null;
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, [queryClient]);

  const persist = useCallback(async (session: AuthSession) => {
    const tokens = { accessToken: session.accessToken, refreshToken: session.refreshToken };
    tokensRef.current = tokens;
    await saveTokens(tokens);
    setUser(session.user);
  }, []);

  const refresh = useCallback(async (): Promise<string | null> => {
    const current = tokensRef.current;
    if (!current) {
      return null;
    }
    try {
      const session = await authApi.refresh(current.refreshToken);
      await persist(session);
      return session.accessToken;
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        await forgetSession();
      }
      return null;
    }
  }, [forgetSession, persist]);

  useEffect(() => {
    setSessionBridge({ getAccessToken: () => tokensRef.current?.accessToken ?? null, refresh });
    return () => setSessionBridge(null);
  }, [refresh]);

  useEffect(() => {
    let active = true;
    (async () => {
      const stored = await loadTokens();
      if (!active) {
        return;
      }
      if (!stored) {
        setStatus("signedOut");
        return;
      }
      tokensRef.current = stored;
      try {
        const me = await authApi.me();
        if (active) {
          setUser(me);
          setStatus("signedIn");
        }
      } catch {
        if (active) {
          await forgetSession();
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [forgetSession]);

  const signIn = useCallback(
    async (session: AuthSession) => {
      queryClient.clear();
      await persist(session);
      setStatus("signedIn");
    },
    [persist, queryClient]
  );

  const signOut = useCallback(async () => {
    const current = tokensRef.current;
    if (current) {
      authApi.logout(current.refreshToken).catch(() => undefined);
    }
    await forgetSession();
  }, [forgetSession]);

  const value = useMemo<AuthState>(
    () => ({ status, user, signIn, signOut, forgetSession }),
    [forgetSession, signIn, signOut, status, user]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return ctx;
}
