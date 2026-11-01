import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "../api";
import { setSessionExpiredHandler, setTokenRefreshedHandler } from "../api/client";
import { clearTokens, getTokens, saveTokens } from "../api/tokenStore";
import type { AccountUser, AuthResponse } from "../api/types";

type AuthState = {
  user: AccountUser | null;
  accessToken: string | null;
  isBooting: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  deleteAccount: (password: string) => Promise<void>;
  /** Recharge /me (ex: apres la creation du profil). */
  refreshUser: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [user, setUser] = useState<AccountUser | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [isBooting, setIsBooting] = useState(true);

  const reset = useCallback(() => {
    setUser(null);
    setAccessToken(null);
    queryClient.clear();
  }, [queryClient]);

  const applySession = useCallback(async (response: AuthResponse) => {
    await saveTokens({ accessToken: response.accessToken, refreshToken: response.refreshToken });
    setAccessToken(response.accessToken);
    setUser(response.user);
  }, []);

  // Garde le token React a jour quand le client HTTP le renouvelle.
  const syncToken = useCallback(async () => {
    const tokens = await getTokens();
    setAccessToken(tokens?.accessToken ?? null);
  }, []);

  useEffect(() => {
    setSessionExpiredHandler(reset);
    setTokenRefreshedHandler(setAccessToken);
    return () => {
      setSessionExpiredHandler(null);
      setTokenRefreshedHandler(null);
    };
  }, [reset]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const tokens = await getTokens();
        if (tokens) {
          const me = await authApi.me();
          if (!cancelled) {
            setUser(me);
            await syncToken();
          }
        }
      } catch {
        // Session invalide ou reseau indisponible: retour a l'ecran de connexion.
      } finally {
        if (!cancelled) {
          setIsBooting(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [syncToken]);

  const signIn = useCallback(
    async (email: string, password: string) => {
      await applySession(await authApi.login(email.trim().toLowerCase(), password));
    },
    [applySession],
  );

  const signUp = useCallback(
    async (email: string, password: string) => {
      await applySession(await authApi.register(email.trim().toLowerCase(), password));
    },
    [applySession],
  );

  const signOut = useCallback(async () => {
    const tokens = await getTokens();
    if (tokens) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // la deconnexion locale prime
      }
    }
    await clearTokens();
    reset();
  }, [reset]);

  const deleteAccount = useCallback(
    async (password: string) => {
      await authApi.deleteAccount(password);
      await clearTokens();
      reset();
    },
    [reset],
  );

  const refreshUser = useCallback(async () => {
    setUser(await authApi.me());
    await syncToken();
  }, [syncToken]);

  const value = useMemo(
    () => ({ user, accessToken, isBooting, signIn, signUp, signOut, deleteAccount, refreshUser }),
    [user, accessToken, isBooting, signIn, signUp, signOut, deleteAccount, refreshUser],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
