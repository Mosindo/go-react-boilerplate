import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { api, ApiError, onSessionExpired } from "../api/client";
import { authApi } from "../api/endpoints";
import type { AuthSession, AuthUser } from "../api/types";
import { clearTokens, currentTokens, loadTokens, saveTokens } from "./tokenStore";
import { resetSwipedMemory } from "../features/discover/swipedMemory";

type Status = "booting" | "signedOut" | "signedIn";

type AuthValue = {
  status: Status;
  user: AuthUser | null;
  bootError: string | null;
  retryBoot: () => void;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
};

const AuthContext = createContext<AuthValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [bootError, setBootError] = useState<string | null>(null);
  const [bootNonce, setBootNonce] = useState(0);

  const localSignOut = useCallback(async () => {
    await clearTokens();
    resetSwipedMemory();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, [queryClient]);

  useEffect(() => {
    onSessionExpired(() => {
      void localSignOut();
    });
    return () => onSessionExpired(null);
  }, [localSignOut]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setBootError(null);
      const tokens = await loadTokens();
      if (!tokens) {
        if (!cancelled) setStatus("signedOut");
        return;
      }
      try {
        // Any authenticated call validates (and if needed refreshes) the stored session.
        const me = await api<AuthUser>("/me");
        if (!cancelled) {
          setUser(me);
          setStatus("signedIn");
        }
      } catch (err) {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 0) {
          setBootError(err.message);
        } else {
          await localSignOut();
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [bootNonce, localSignOut]);

  const signIn = useCallback(
    async (session: AuthSession) => {
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      queryClient.clear();
      resetSwipedMemory();
      setUser(session.user);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const signOut = useCallback(async () => {
    const refresh = currentTokens()?.refreshToken;
    if (refresh) {
      try {
        await authApi.logout(refresh);
      } catch {
        // Offline logout still clears the local session.
      }
    }
    await localSignOut();
  }, [localSignOut]);

  const value = useMemo<AuthValue>(
    () => ({ status, user, bootError, retryBoot: () => setBootNonce((n) => n + 1), signIn, signOut }),
    [status, user, bootError, signIn, signOut]
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
