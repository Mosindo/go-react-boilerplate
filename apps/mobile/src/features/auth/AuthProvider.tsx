import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError, configureSession } from "../../api/client";
import { authApi } from "../../api/endpoints";
import type { AuthResponse, AuthUser } from "../../api/types";
import { clearTokens, getTokens, saveTokens } from "../../store/tokenStore";
import { showToast } from "../../ui";

type AuthStatus = "booting" | "offline" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: AuthStatus;
  user: AuthUser | null;
  accessToken: string | null;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears the local session without calling the API (after account deletion). */
  forgetSession: () => Promise<void>;
  retryBoot: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [bootAttempt, setBootAttempt] = useState(0);

  const forgetSession = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setAccessToken(null);
    setStatus("signedOut");
  }, [queryClient]);

  useEffect(() => {
    configureSession({
      onTokens: (tokens) => setAccessToken(tokens.accessToken),
      onExpired: () => {
        void forgetSession();
        showToast("Votre session a expiré. Reconnectez-vous.", "error");
      }
    });
    return () => configureSession(null);
  }, [forgetSession]);

  useEffect(() => {
    let active = true;
    (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.me();
        const latest = await getTokens();
        if (!active) return;
        setUser(me);
        setAccessToken(latest?.accessToken ?? tokens.accessToken);
        setStatus("signedIn");
      } catch (err) {
        if (!active) return;
        if (err instanceof ApiError && err.status === 0) {
          setStatus("offline");
        } else {
          await clearTokens();
          setStatus("signedOut");
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [bootAttempt]);

  const applySession = useCallback(
    async (response: AuthResponse) => {
      await saveTokens({ accessToken: response.accessToken, refreshToken: response.refreshToken });
      queryClient.clear();
      setUser(response.user);
      setAccessToken(response.accessToken);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const signIn = useCallback(
    async (email: string, password: string) => {
      await applySession(await authApi.login(email.trim().toLowerCase(), password));
    },
    [applySession]
  );

  const signUp = useCallback(
    async (email: string, password: string) => {
      await applySession(await authApi.register(email.trim().toLowerCase(), password));
    },
    [applySession]
  );

  const signOut = useCallback(async () => {
    const tokens = await getTokens();
    if (tokens?.refreshToken) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // Local sign-out still proceeds.
      }
    }
    await forgetSession();
  }, [forgetSession]);

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      user,
      accessToken,
      signIn,
      signUp,
      signOut,
      forgetSession,
      retryBoot: () => {
        setStatus("booting");
        setBootAttempt((n) => n + 1);
      }
    }),
    [status, user, accessToken, signIn, signUp, signOut, forgetSession]
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
