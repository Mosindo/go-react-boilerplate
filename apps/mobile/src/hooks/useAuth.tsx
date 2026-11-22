import React, { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { logout as logoutRequest, me } from "../api/auth";
import { ApiError, onSessionExpired } from "../api/client";
import type { AuthSession, AuthUser } from "../api/types";
import { clearTokens, getTokens, loadTokens, saveTokens } from "../store/tokenStore";
import { showToast } from "../ui";

type AuthStatus = "booting" | "offline" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: AuthStatus;
  user: AuthUser | null;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  /** Local sign-out without calling the API (account deleted, session gone). */
  forgetSession: () => Promise<void>;
  retryBoot: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [bootKey, setBootKey] = useState(0);

  const forgetSession = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, [queryClient]);

  useEffect(() => {
    let active = true;
    (async () => {
      const tokens = await loadTokens();
      if (!tokens) {
        if (active) {
          setStatus("signedOut");
        }
        return;
      }
      try {
        const current = await me();
        if (active) {
          setUser(current);
          setStatus("signedIn");
        }
      } catch (err) {
        if (!active) {
          return;
        }
        if (err instanceof ApiError && (err.status === 401 || err.status === 404)) {
          await forgetSession();
        } else {
          setStatus("offline");
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [bootKey, forgetSession]);

  useEffect(
    () =>
      onSessionExpired(() => {
        queryClient.clear();
        setUser(null);
        setStatus("signedOut");
        showToast("Votre session a expiré. Reconnectez-vous.", "error");
      }),
    [queryClient]
  );

  const signIn = useCallback(
    async (session: AuthSession) => {
      queryClient.clear();
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      setUser(session.user);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const signOut = useCallback(async () => {
    const refreshToken = getTokens()?.refreshToken;
    if (refreshToken) {
      try {
        await logoutRequest(refreshToken);
      } catch {
        // Best effort: local sign-out always wins.
      }
    }
    await forgetSession();
  }, [forgetSession]);

  const value = useMemo<AuthContextValue>(
    () => ({ status, user, signIn, signOut, forgetSession, retryBoot: () => setBootKey((k) => k + 1) }),
    [forgetSession, signIn, signOut, status, user]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return ctx;
}

/** The signed-in member; only call below the signed-in navigator. */
export function useCurrentUser(): AuthUser {
  const { user } = useAuth();
  if (!user) {
    throw new Error("useCurrentUser called without a session");
  }
  return user;
}
