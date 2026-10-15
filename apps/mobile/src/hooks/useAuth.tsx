import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  login as loginRequest,
  logout as logoutRequest,
  me,
  register as registerRequest,
  type AuthSession,
  type AuthUser
} from "../api/auth";
import { ApiError, setSessionExpiredHandler } from "../api/client";
import { showToast } from "../shared/feedback";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";

type AuthStatus = "booting" | "signedOut" | "signedIn" | "offline";

type AuthContextValue = {
  status: AuthStatus;
  user: AuthUser | null;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state after the server deleted the account. */
  accountDeleted: () => Promise<void>;
  /** Retries restoring the session after a network failure at startup. */
  retry: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [bootAttempt, setBootAttempt] = useState(0);

  const endSession = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, [queryClient]);

  const applySession = useCallback(async (session: AuthSession) => {
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    setUser(session.user);
    setStatus("signedIn");
  }, []);

  // Restore the stored session on launch.
  useEffect(() => {
    let active = true;
    (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const current = await me();
        if (active) {
          setUser(current);
          setStatus("signedIn");
        }
      } catch (error) {
        if (!active) return;
        if (error instanceof ApiError && error.status === 0) {
          setStatus("offline");
        } else {
          await endSession();
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [bootAttempt, endSession]);

  // The API client reports unrecoverable 401s (refresh token revoked or expired).
  useEffect(() => {
    setSessionExpiredHandler(() => {
      queryClient.clear();
      setUser(null);
      setStatus((current) => {
        if (current === "signedIn") {
          showToast({ message: "Votre session a expiré, reconnectez-vous.", tone: "danger" });
        }
        return "signedOut";
      });
    });
    return () => setSessionExpiredHandler(null);
  }, [queryClient]);

  const signIn = useCallback(
    async (email: string, password: string) => {
      await applySession(await loginRequest(email, password));
    },
    [applySession]
  );

  const signUp = useCallback(
    async (email: string, password: string) => {
      await applySession(await registerRequest(email, password));
    },
    [applySession]
  );

  const signOut = useCallback(async () => {
    const tokens = await getTokens();
    try {
      if (tokens) {
        await logoutRequest(tokens.refreshToken);
      }
    } catch {
      // Signing out locally always wins; the server session expires on its own.
    }
    await endSession();
  }, [endSession]);

  const retry = useCallback(() => {
    setStatus("booting");
    setBootAttempt((n) => n + 1);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({ status, user, signIn, signUp, signOut, accountDeleted: endSession, retry }),
    [endSession, retry, signIn, signOut, signUp, status, user]
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
