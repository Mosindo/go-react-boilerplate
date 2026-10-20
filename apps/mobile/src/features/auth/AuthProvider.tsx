import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchMe, logout as logoutRequest } from "../../api/auth";
import { ApiError, setAuthLostHandler } from "../../api/client";
import { queryKeys } from "../../api/keys";
import type { AuthSession, User } from "../../api/types";
import { clearTokens, getTokens, saveTokens } from "../../store/tokenStore";

type AuthValue = {
  status: "booting" | "anonymous" | "authenticated";
  user: User | null;
  signIn: (session: AuthSession) => Promise<void>;
  signOut: () => Promise<void>;
  refreshUser: () => Promise<void>;
};

const AuthContext = createContext<AuthValue | null>(null);

export function useAuth(): AuthValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [hasSession, setHasSession] = useState<boolean | null>(null);

  useEffect(() => {
    let active = true;
    void getTokens().then((tokens) => {
      if (active) setHasSession(Boolean(tokens));
    });
    return () => {
      active = false;
    };
  }, []);

  const localSignOut = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setHasSession(false);
  }, [queryClient]);

  useEffect(() => {
    setAuthLostHandler(() => void localSignOut());
    return () => setAuthLostHandler(null);
  }, [localSignOut]);

  const meQuery = useQuery({
    queryKey: queryKeys.me,
    queryFn: fetchMe,
    enabled: hasSession === true,
    retry: (count, error) => !(error instanceof ApiError && error.status === 401) && count < 2
  });

  useEffect(() => {
    if (meQuery.error instanceof ApiError && meQuery.error.status === 401) void localSignOut();
  }, [meQuery.error, localSignOut]);

  const signIn = useCallback(
    async (session: AuthSession) => {
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      queryClient.setQueryData(queryKeys.me, session.user);
      setHasSession(true);
    },
    [queryClient]
  );

  const signOut = useCallback(async () => {
    const tokens = await getTokens();
    if (tokens) {
      try {
        await logoutRequest(tokens.refreshToken);
      } catch {
        // Server-side revocation is best effort; local tokens are always cleared.
      }
    }
    await localSignOut();
  }, [localSignOut]);

  const refreshUser = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: queryKeys.me });
  }, [queryClient]);

  const value = useMemo<AuthValue>(() => {
    const user = meQuery.data ?? null;
    let status: AuthValue["status"] = "anonymous";
    if (hasSession === null || (hasSession && !user && !meQuery.isError)) status = "booting";
    else if (hasSession && user) status = "authenticated";
    return { status, user, signIn, signOut, refreshUser };
  }, [hasSession, meQuery.data, meQuery.isError, signIn, signOut, refreshUser]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
