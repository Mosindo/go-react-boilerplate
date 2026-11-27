import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { ApiError, onSessionExpired } from "../api/client";
import { authApi } from "../api/endpoints";
import { realtime } from "../api/realtime";
import type { AuthSession, AuthUser } from "../api/types";
import { clearTokens, getTokens, restoreTokens, saveTokens } from "../store/tokenStore";
import { toast } from "../ui";
import { queryClient } from "./queryClient";

type AuthStatus = "booting" | "signedOut" | "signedIn" | "offline";

type AuthValue = {
  status: AuthStatus;
  user: AuthUser | null;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: (options?: { remote?: boolean }) => Promise<void>;
  retryBoot: () => void;
};

const AuthContext = createContext<AuthValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);
  const [bootKey, setBootKey] = useState(0);

  const resetLocal = useCallback(async () => {
    realtime.stop();
    await clearTokens();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, []);

  useEffect(() => {
    onSessionExpired(() => {
      void resetLocal();
      toast("Ta session a expiré, reconnecte-toi.", "info");
    });
  }, [resetLocal]);

  useEffect(() => {
    let active = true;
    (async () => {
      const tokens = await restoreTokens();
      if (!tokens) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.me();
        if (!active) return;
        setUser(me);
        setStatus("signedIn");
      } catch (error) {
        if (!active) return;
        if (error instanceof ApiError && error.status === 401) {
          await resetLocal();
        } else {
          setStatus("offline");
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [bootKey, resetLocal]);

  const signIn = useCallback(async (session: AuthSession) => {
    queryClient.clear();
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    setUser(session.user);
    setStatus("signedIn");
  }, []);

  const signOut = useCallback(
    async ({ remote = true }: { remote?: boolean } = {}) => {
      const refreshToken = getTokens()?.refreshToken;
      if (remote && refreshToken) {
        try {
          await authApi.logout(refreshToken);
        } catch {
          // Best effort: local sign-out always wins.
        }
      }
      await resetLocal();
    },
    [resetLocal]
  );

  const retryBoot = useCallback(() => {
    setStatus("booting");
    setBootKey((k) => k + 1);
  }, []);

  const value = useMemo(() => ({ status, user, signIn, signOut, retryBoot }), [status, user, signIn, signOut, retryBoot]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return value;
}

/** The signed-in user id (screens below the auth gate only). */
export function useMyId(): string {
  return useAuth().user?.id ?? "";
}
