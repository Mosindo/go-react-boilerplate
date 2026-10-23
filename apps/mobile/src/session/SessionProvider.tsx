import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import {
  getMe,
  http,
  logout as logoutRequest,
  onSessionExpired,
  type AuthResponse,
  type Me
} from "../api";
import { t } from "../i18n/fr";
import { tokenStorage } from "../store/tokenStore";
import { useToast } from "../ui";
import { queryKeys } from "./queryKeys";

type Status = "booting" | "signedOut" | "signedIn";

type SessionValue = {
  status: Status;
  me: UseQueryResult<Me, Error>;
  userId: string | null;
  signIn: (auth: AuthResponse) => Promise<void>;
  signOut: () => Promise<void>;
  /** Drops the local session without calling the API (account deleted, session expired). */
  endLocalSession: () => Promise<void>;
};

const SessionContext = createContext<SessionValue | null>(null);

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [status, setStatus] = useState<Status>("booting");

  useEffect(() => {
    let active = true;
    void tokenStorage.get().then((tokens) => {
      if (!active) return;
      http.setCachedAccessToken(tokens?.accessToken ?? null);
      setStatus(tokens ? "signedIn" : "signedOut");
    });
    return () => {
      active = false;
    };
  }, []);

  const endLocalSession = useCallback(async () => {
    await tokenStorage.clear();
    http.setCachedAccessToken(null);
    queryClient.clear();
    setStatus("signedOut");
  }, [queryClient]);

  useEffect(
    () =>
      onSessionExpired(() => {
        void endLocalSession();
        toast.error(t.errors.sessionExpired);
      }),
    [endLocalSession, toast]
  );

  const signIn = useCallback(
    async (auth: AuthResponse) => {
      queryClient.clear();
      await tokenStorage.set({ accessToken: auth.accessToken, refreshToken: auth.refreshToken });
      http.setCachedAccessToken(auth.accessToken);
      setStatus("signedIn");
    },
    [queryClient]
  );

  const signOut = useCallback(async () => {
    try {
      const tokens = await tokenStorage.get();
      if (tokens) await logoutRequest(tokens.refreshToken);
    } catch {
      // best effort: the local session is dropped regardless
    }
    await endLocalSession();
  }, [endLocalSession]);

  const me = useQuery({
    queryKey: queryKeys.me,
    queryFn: getMe,
    enabled: status === "signedIn",
    retry: false
  });

  const value = useMemo<SessionValue>(
    () => ({ status, me, userId: me.data?.id ?? null, signIn, signOut, endLocalSession }),
    [status, me, signIn, signOut, endLocalSession]
  );
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionValue {
  const value = useContext(SessionContext);
  if (!value) throw new Error("useSession must be used within SessionProvider");
  return value;
}
