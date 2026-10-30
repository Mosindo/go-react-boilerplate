import React, { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { authApi } from "../api/endpoints";
import { setSessionExpiredHandler } from "../api/client";
import type { AuthUser, Session } from "../api/types";
import { clearTokens, loadTokens, saveTokens } from "../store/tokenStore";
import { discoverStore } from "./discoverStore";
import { queryClient } from "./queryClient";

type Status = "booting" | "signedOut" | "signedIn";

type SessionValue = {
  status: Status;
  user: AuthUser | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  /** Drops local state after the server already removed the account. */
  endSession: () => Promise<void>;
};

const SessionContext = createContext<SessionValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  const clearLocal = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    discoverStore.reset();
    setUser(null);
    setStatus("signedOut");
  }, []);

  const adopt = useCallback(async (session: Session) => {
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    queryClient.clear();
    discoverStore.reset();
    setUser(session.user);
    setStatus("signedIn");
  }, []);

  useEffect(() => {
    setSessionExpiredHandler(() => {
      queryClient.clear();
      discoverStore.reset();
      setUser(null);
      setStatus("signedOut");
    });

    let active = true;
    (async () => {
      const tokens = await loadTokens();
      if (!tokens) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.me();
        if (active) {
          setUser(me);
          setStatus("signedIn");
        }
      } catch {
        if (active) await clearLocal();
      }
    })();
    return () => {
      active = false;
    };
  }, [clearLocal]);

  const login = useCallback(
    async (email: string, password: string) => adopt(await authApi.login(email, password)),
    [adopt]
  );
  const register = useCallback(
    async (email: string, password: string) => adopt(await authApi.register(email, password)),
    [adopt]
  );

  const logout = useCallback(async () => {
    const tokens = await loadTokens();
    if (tokens) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // Local sign-out must succeed even when the server is unreachable.
      }
    }
    await clearLocal();
  }, [clearLocal]);

  const value = useMemo(
    () => ({ status, user, login, register, logout, endSession: clearLocal }),
    [status, user, login, register, logout, clearLocal]
  );
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionValue {
  const value = useContext(SessionContext);
  if (!value) {
    throw new Error("useSession must be used inside SessionProvider");
  }
  return value;
}
