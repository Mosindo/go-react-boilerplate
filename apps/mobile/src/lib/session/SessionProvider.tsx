import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { authApi } from "../api/endpoints";
import { onSessionExpired } from "../api/client";
import { session } from "../api/session";
import type { AuthSession } from "../api/types";
import { queryClient } from "../queryClient";

type Status = "restoring" | "signedOut" | "signedIn";

type SessionContextValue = {
  status: Status;
  userId: string | null;
  isModerator: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state after the account was deleted server-side. */
  forget: () => Promise<void>;
  expiredNotice: boolean;
  dismissExpiredNotice: () => void;
};

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<Status>("restoring");
  const [userId, setUserId] = useState<string | null>(null);
  const [isModerator, setIsModerator] = useState(false);
  const [expiredNotice, setExpiredNotice] = useState(false);

  const reset = useCallback(async () => {
    await session.set(null);
    queryClient.clear();
    setUserId(null);
    setIsModerator(false);
    setStatus("signedOut");
  }, []);

  useEffect(() => {
    onSessionExpired(() => {
      queryClient.clear();
      setUserId(null);
      setIsModerator(false);
      setStatus("signedOut");
      setExpiredNotice(true);
    });
    let active = true;
    (async () => {
      const tokens = await session.restore();
      if (!tokens) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.me();
        if (active) {
          setUserId(me.id);
          setIsModerator(me.role === "moderator");
          setStatus("signedIn");
        }
      } catch {
        if (active) await reset();
      }
    })();
    return () => {
      active = false;
    };
  }, [reset]);

  const apply = useCallback(async (auth: AuthSession) => {
    queryClient.clear();
    await session.set({ accessToken: auth.accessToken, refreshToken: auth.refreshToken });
    setExpiredNotice(false);
    setUserId(auth.user.id);
    setIsModerator(auth.user.role === "moderator");
    setStatus("signedIn");
  }, []);

  const value = useMemo<SessionContextValue>(
    () => ({
      status,
      userId,
      isModerator,
      expiredNotice,
      dismissExpiredNotice: () => setExpiredNotice(false),
      signIn: async (email, password) => apply(await authApi.login(email.trim(), password)),
      signUp: async (email, password) => apply(await authApi.register(email.trim(), password)),
      signOut: async () => {
        const refreshToken = session.get()?.refreshToken;
        if (refreshToken) {
          await authApi.logout(refreshToken).catch(() => undefined);
        }
        await reset();
      },
      forget: reset
    }),
    [apply, expiredNotice, isModerator, reset, status, userId]
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionContextValue {
  const ctx = useContext(SessionContext);
  if (!ctx) {
    throw new Error("useSession must be used inside SessionProvider");
  }
  return ctx;
}
