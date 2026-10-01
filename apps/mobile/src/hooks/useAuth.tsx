import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { logout as logoutRequest, me } from "../api/auth";
import { ApiError, setAuthBridge } from "../api/client";
import type { AuthSession, AuthUser } from "../api/types";
import { clearTokens, getTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

type AuthContextValue = {
  /** True until the stored session has been restored (or found missing). */
  isBooting: boolean;
  isAuthenticated: boolean;
  accessToken: string | null;
  user: AuthUser | null;
  /** Set when the session ended on its own (expired, suspended). */
  sessionNotice: string | null;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Re-reads /me (e.g. after the profile became complete). */
  refreshUser: () => Promise<AuthUser | null>;
  clearSessionNotice: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [tokens, setTokens] = useState<AuthTokens | null>(null);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [isBooting, setBooting] = useState(true);
  const [sessionNotice, setSessionNotice] = useState<string | null>(null);
  const tokensRef = useRef<AuthTokens | null>(null);

  const applyTokens = useCallback(async (next: AuthTokens | null) => {
    tokensRef.current = next;
    setTokens(next);
    if (next) {
      await saveTokens(next);
    } else {
      await clearTokens();
    }
  }, []);

  const endSession = useCallback(
    async (notice: string | null) => {
      await applyTokens(null);
      setUser(null);
      queryClient.clear();
      setSessionNotice(notice);
    },
    [applyTokens, queryClient]
  );

  // The API client refreshes tokens transparently; it reports back through this bridge.
  useEffect(() => {
    setAuthBridge({
      getTokens: async () => tokensRef.current ?? (await getTokens()),
      setTokens: async (next, nextUser) => {
        await applyTokens(next);
        setUser(nextUser);
      },
      onAuthLost: () => {
        void endSession("Votre session a expiré. Reconnectez-vous.");
      }
    });
    return () => setAuthBridge(null);
  }, [applyTokens, endSession]);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const stored = await getTokens();
        if (!stored) {
          return;
        }
        tokensRef.current = stored;
        setTokens(stored);
        const current = await me();
        if (active) {
          setUser(current);
        }
      } catch (error) {
        // only a definitive auth failure ends the session; a network blip keeps the tokens
        if (active && error instanceof ApiError && (error.status === 401 || error.status === 403)) {
          await endSession(error.status === 403 ? error.message : null);
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

  const signIn = useCallback(
    async (session: AuthSession) => {
      await applyTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      setUser(session.user);
      setSessionNotice(null);
    },
    [applyTokens]
  );

  const signOut = useCallback(async () => {
    const refreshToken = tokensRef.current?.refreshToken;
    if (refreshToken) {
      try {
        await logoutRequest(refreshToken);
      } catch {
        // best effort: local sign-out always wins
      }
    }
    await endSession(null);
  }, [endSession]);

  const refreshUser = useCallback(async () => {
    try {
      const current = await me();
      setUser(current);
      return current;
    } catch {
      return null;
    }
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      isBooting,
      isAuthenticated: Boolean(tokens && user),
      accessToken: tokens?.accessToken ?? null,
      user,
      sessionNotice,
      signIn,
      signOut,
      refreshUser,
      clearSessionNotice: () => setSessionNotice(null)
    }),
    [isBooting, tokens, user, sessionNotice, signIn, signOut, refreshUser]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
