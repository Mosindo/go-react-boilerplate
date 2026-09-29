import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { deleteAccount as deleteAccountRequest, getMe, type Me } from "../api/account";
import {
  login as loginRequest,
  logout as logoutRequest,
  register as registerRequest,
  type AuthSession
} from "../api/auth";
import { registerAuthHandlers } from "../api/client";
import { messageFromError } from "../lib/errors";
import { strings } from "../lib/strings";
import { normalizeEmail } from "../lib/validation";
import { clearTokens, getTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

type AuthContextValue = {
  user: Me | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isBooting: boolean;
  /** Set when the stored session could not be verified because of a network/server problem. */
  bootError: string | null;
  /** Shown on the sign-in screens, e.g. after the session expired. */
  authNotice: string | null;
  signIn: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, birthDate: string) => Promise<void>;
  signOut: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
  /** Re-reads GET /me (profileComplete, age...). */
  refreshProfileState: () => Promise<Me | null>;
  retryBoot: () => void;
  clearAuthNotice: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [tokens, setTokens] = useState<AuthTokens | null>(null);
  const [user, setUser] = useState<Me | null>(null);
  const [isBooting, setIsBooting] = useState(true);
  const [bootError, setBootError] = useState<string | null>(null);
  const [authNotice, setAuthNotice] = useState<string | null>(null);
  const [bootAttempt, setBootAttempt] = useState(0);
  const expiredRef = useRef(false);

  const resetSession = useCallback(() => {
    setTokens(null);
    setUser(null);
    queryClient.clear();
  }, [queryClient]);

  // The API client refreshes tokens and reports expiry; keep React state in sync with it.
  useEffect(
    () =>
      registerAuthHandlers({
        onTokensRefreshed: (next) => setTokens(next),
        onSessionExpired: () => {
          expiredRef.current = true;
          resetSession();
          setAuthNotice(strings.errors.unauthorized);
        }
      }),
    [resetSession]
  );

  // Session restore.
  useEffect(() => {
    let active = true;
    async function boot() {
      setIsBooting(true);
      setBootError(null);
      expiredRef.current = false;
      try {
        const stored = await getTokens();
        if (!stored) {
          return;
        }
        setTokens(stored);
        const me = await getMe();
        if (active) {
          setUser(me);
        }
      } catch (error) {
        if (!active || expiredRef.current) {
          return;
        }
        setBootError(messageFromError(error, strings.errors.network));
      } finally {
        if (active) {
          setIsBooting(false);
        }
      }
    }
    void boot();
    return () => {
      active = false;
    };
  }, [bootAttempt]);

  const applySession = useCallback(async (session: AuthSession) => {
    const next: AuthTokens = { accessToken: session.accessToken, refreshToken: session.refreshToken };
    await saveTokens(next);
    expiredRef.current = false;
    setTokens(next);
    setUser(session.user);
    setAuthNotice(null);
    setBootError(null);
  }, []);

  const signIn = useCallback(
    async (email: string, password: string) => {
      await applySession(await loginRequest(normalizeEmail(email), password));
    },
    [applySession]
  );

  const register = useCallback(
    async (email: string, password: string, birthDate: string) => {
      await applySession(await registerRequest(normalizeEmail(email), password, birthDate));
    },
    [applySession]
  );

  const signOut = useCallback(async () => {
    const stored = await getTokens();
    try {
      if (stored?.refreshToken) {
        await logoutRequest(stored.refreshToken);
      }
    } catch {
      // Best effort: local cleanup always wins.
    }
    await clearTokens();
    resetSession();
    setAuthNotice(null);
  }, [resetSession]);

  const deleteAccount = useCallback(
    async (password: string) => {
      await deleteAccountRequest(password);
      await clearTokens();
      resetSession();
      setAuthNotice("Your account and all its data have been deleted.");
    },
    [resetSession]
  );

  const refreshProfileState = useCallback(async () => {
    const me = await getMe();
    setUser(me);
    return me;
  }, []);

  const retryBoot = useCallback(() => setBootAttempt((value) => value + 1), []);
  const clearAuthNotice = useCallback(() => setAuthNotice(null), []);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      accessToken: tokens?.accessToken ?? null,
      isAuthenticated: Boolean(tokens?.accessToken && user),
      isBooting,
      bootError,
      authNotice,
      signIn,
      register,
      signOut,
      deleteAccount,
      refreshProfileState,
      retryBoot,
      clearAuthNotice
    }),
    [
      authNotice,
      bootError,
      clearAuthNotice,
      deleteAccount,
      isBooting,
      refreshProfileState,
      register,
      retryBoot,
      signIn,
      signOut,
      tokens?.accessToken,
      user
    ]
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
