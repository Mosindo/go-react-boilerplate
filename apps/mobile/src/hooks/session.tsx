import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "../api/endpoints";
import { isApiError, setSessionLostHandler } from "../api/client";
import { tokenStore } from "../api/tokenStore";
import type { AuthResponse, User } from "../api/types";

type Status = "booting" | "signedOut" | "signedIn" | "bootError";

type SessionValue = {
  status: Status;
  user: User | null;
  retryBoot: () => void;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Local cleanup after the server already invalidated the session (account deletion). */
  dropSession: () => Promise<void>;
};

const SessionContext = createContext<SessionValue | null>(null);

export function useSession(): SessionValue {
  const value = useContext(SessionContext);
  if (!value) throw new Error("useSession must be used inside SessionProvider");
  return value;
}

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<User | null>(null);

  const dropSession = useCallback(async () => {
    await tokenStore.clear();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, [queryClient]);

  useEffect(() => {
    setSessionLostHandler(() => {
      void dropSession();
    });
  }, [dropSession]);

  const boot = useCallback(async () => {
    setStatus("booting");
    const tokens = await tokenStore.get();
    if (!tokens) {
      setStatus("signedOut");
      return;
    }
    try {
      const me = await authApi.me();
      setUser(me);
      setStatus("signedIn");
    } catch (err) {
      if (isApiError(err) && err.status === 401) {
        await dropSession();
      } else {
        setStatus("bootError");
      }
    }
  }, [dropSession]);

  useEffect(() => {
    void boot();
  }, [boot]);

  const adopt = useCallback(async (res: AuthResponse) => {
    await tokenStore.set({ accessToken: res.accessToken, refreshToken: res.refreshToken });
    queryClient.clear();
    setUser(res.user);
    setStatus("signedIn");
  }, [queryClient]);

  const value = useMemo<SessionValue>(
    () => ({
      status,
      user,
      retryBoot: () => void boot(),
      signIn: async (email, password) => adopt(await authApi.login(email.trim(), password)),
      signUp: async (email, password) => adopt(await authApi.register(email.trim(), password)),
      signOut: async () => {
        const tokens = await tokenStore.get();
        if (tokens) {
          try {
            await authApi.logout(tokens.refreshToken);
          } catch {
            // the local session is dropped regardless
          }
        }
        await dropSession();
      },
      dropSession,
    }),
    [status, user, boot, adopt, dropSession],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}
