import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import {
  deleteAccount as deleteAccountRequest,
  login as loginRequest,
  logout as logoutRequest,
  me,
  register as registerRequest
} from "../api/auth";
import { ApiError, setSessionExpiredHandler } from "../api/client";
import type { AuthSession, User } from "../api/types";
import { strings } from "../i18n/strings";
import { resetFeedback, showToast } from "../shared/feedback";
import { queryClient } from "../store/queryClient";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import { normalizeEmail } from "../utils/validation";

type AuthStatus = "booting" | "signedOut" | "signedIn";

export type AuthContextValue = {
  status: AuthStatus;
  isBooting: boolean;
  isAuthenticated: boolean;
  user: User | null;
  /** Set when the stored session could not be verified because the network/server failed. */
  bootError: string | null;
  retryBoot: () => void;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  /** Re-reads GET /me (e.g. after the profile or its first photo was created). */
  refreshUser: () => Promise<User>;
  logout: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

/** Gate rule: signed in users without a complete profile must go through onboarding. */
export function needsOnboarding(user: User | null): boolean {
  return !!user && (!user.hasProfile || !user.profileComplete);
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<User | null>(null);
  const [bootError, setBootError] = useState<string | null>(null);
  const [bootAttempt, setBootAttempt] = useState(0);

  const clearLocalSession = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    resetFeedback();
    setUser(null);
    setStatus("signedOut");
  }, []);

  useEffect(() => {
    setSessionExpiredHandler(() => {
      queryClient.clear();
      setUser(null);
      setStatus("signedOut");
      showToast(strings.auth.sessionExpired, "info");
    });
    return () => setSessionExpiredHandler(null);
  }, []);

  useEffect(() => {
    let active = true;
    setStatus("booting");
    setBootError(null);

    async function bootstrap() {
      const tokens = await getTokens();
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
      } catch (error) {
        if (!active) {
          return;
        }
        if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
          await clearLocalSession();
        } else {
          setBootError(error instanceof Error ? error.message : strings.errors.network);
        }
      }
    }

    void bootstrap();
    return () => {
      active = false;
    };
  }, [bootAttempt, clearLocalSession]);

  const applySession = useCallback(async (session: AuthSession) => {
    queryClient.clear();
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    setUser(session.user);
    setStatus("signedIn");
  }, []);

  const login = useCallback(
    async (email: string, password: string) => {
      await applySession(await loginRequest(normalizeEmail(email), password));
    },
    [applySession]
  );

  const register = useCallback(
    async (email: string, password: string) => {
      await applySession(await registerRequest(normalizeEmail(email), password));
    },
    [applySession]
  );

  const refreshUser = useCallback(async () => {
    const current = await me();
    setUser(current);
    return current;
  }, []);

  const logout = useCallback(async () => {
    const tokens = await getTokens();
    try {
      if (tokens?.refreshToken) {
        await logoutRequest(tokens.refreshToken);
      }
    } catch {
      // Best effort: local cleanup always wins.
    }
    await clearLocalSession();
  }, [clearLocalSession]);

  const deleteAccount = useCallback(
    async (password: string) => {
      await deleteAccountRequest(password);
      await clearLocalSession();
    },
    [clearLocalSession]
  );

  const retryBoot = useCallback(() => setBootAttempt((value) => value + 1), []);

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      isBooting: status === "booting",
      isAuthenticated: status === "signedIn" && user !== null,
      user,
      bootError,
      retryBoot,
      login,
      register,
      refreshUser,
      logout,
      deleteAccount
    }),
    [bootError, deleteAccount, login, logout, refreshUser, register, retryBoot, status, user]
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
