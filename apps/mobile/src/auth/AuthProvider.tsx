import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "../api/endpoints";
import { configureSession, refreshSession, setAccessToken } from "../api/client";
import type { AuthResponse, Me } from "../api/types";
import { clearRefreshToken, loadRefreshToken, saveRefreshToken } from "./tokenStore";

type Status = "booting" | "signedOut" | "signedIn";

type AuthValue = {
  status: Status;
  user: Me | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
  /** Re-reads /me (e.g. after onboarding completes). */
  refreshUser: () => Promise<void>;
};

const AuthContext = createContext<AuthValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<Me | null>(null);
  const refreshTokenRef = useRef<string | null>(null);

  const clearSession = useCallback(async () => {
    refreshTokenRef.current = null;
    setAccessToken(null);
    await clearRefreshToken();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, [queryClient]);

  const applyAuth = useCallback(async (auth: AuthResponse) => {
    refreshTokenRef.current = auth.refreshToken;
    setAccessToken(auth.accessToken);
    await saveRefreshToken(auth.refreshToken);
    setUser(auth.user);
    setStatus("signedIn");
  }, []);

  useEffect(() => {
    configureSession({
      getRefreshToken: async () => refreshTokenRef.current ?? (await loadRefreshToken()),
      onTokens: applyAuth,
      onExpired: clearSession,
    });
    return () => configureSession(null);
  }, [applyAuth, clearSession]);

  // Restore the session on launch from the persisted refresh token.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const stored = await loadRefreshToken();
      if (!stored) {
        if (!cancelled) setStatus("signedOut");
        return;
      }
      refreshTokenRef.current = stored;
      const ok = await refreshSession();
      if (!cancelled && !ok) {
        // Offline at launch keeps the token for the next try; the user sees the sign-in screen.
        setStatus("signedOut");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const value = useMemo<AuthValue>(
    () => ({
      status,
      user,
      login: async (email, password) => applyAuth(await authApi.login(email.trim(), password)),
      register: async (email, password) =>
        applyAuth(await authApi.register(email.trim(), password)),
      logout: async () => {
        const token = refreshTokenRef.current;
        if (token) await authApi.logout(token).catch(() => undefined);
        await clearSession();
      },
      deleteAccount: async (password) => {
        await authApi.deleteAccount(password);
        await clearSession();
      },
      refreshUser: async () => setUser(await authApi.me()),
    }),
    [status, user, applyAuth, clearSession],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
