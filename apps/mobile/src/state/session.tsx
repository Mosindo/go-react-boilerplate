import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, setSessionExpiredListener } from "../api/client";
import { authApi, profileApi } from "../api/endpoints";
import { queryKeys } from "../api/queryKeys";
import { clearTokens, loadTokens, saveTokens } from "../api/tokenStore";
import type { AuthResponse, AuthUser, OwnProfile } from "../api/types";

type Status = "booting" | "signedOut" | "signedIn";

type SessionValue = {
  status: Status;
  user: AuthUser | null;
  /** undefined while loading, null when no profile exists yet. */
  profile: OwnProfile | null | undefined;
  profileError: unknown;
  refetchProfile: () => void;
  signIn: (auth: AuthResponse) => Promise<void>;
  signOut: () => Promise<void>;
  /** Local cleanup after server-side account deletion. */
  forget: () => Promise<void>;
};

const SessionContext = createContext<SessionValue | null>(null);

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const client = useQueryClient();
  const [status, setStatus] = useState<Status>("booting");
  const [user, setUser] = useState<AuthUser | null>(null);

  const reset = useCallback(async () => {
    await clearTokens();
    setUser(null);
    setStatus("signedOut");
    client.clear();
  }, [client]);

  useEffect(() => {
    setSessionExpiredListener(() => {
      void reset();
    });
    return () => setSessionExpiredListener(null);
  }, [reset]);

  useEffect(() => {
    let active = true;
    void (async () => {
      const tokens = await loadTokens();
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
          await reset();
        } else {
          // Offline at boot: keep the session, screens will show retry states.
          setStatus("signedIn");
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [reset]);

  const profileQuery = useQuery({
    queryKey: queryKeys.profile,
    enabled: status === "signedIn",
    queryFn: async () => {
      try {
        return await profileApi.get();
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return null;
        throw error;
      }
    }
  });

  const signIn = useCallback(
    async (auth: AuthResponse) => {
      client.clear();
      await saveTokens({ accessToken: auth.accessToken, refreshToken: auth.refreshToken });
      setUser(auth.user);
      setStatus("signedIn");
    },
    [client]
  );

  const signOut = useCallback(async () => {
    const tokens = await loadTokens();
    if (tokens) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // Local sign-out still proceeds.
      }
    }
    await reset();
  }, [reset]);

  const value = useMemo<SessionValue>(
    () => ({
      status,
      user,
      profile: status === "signedIn" ? profileQuery.data : null,
      profileError: profileQuery.error,
      refetchProfile: () => void profileQuery.refetch(),
      signIn,
      signOut,
      forget: reset
    }),
    [status, user, profileQuery, signIn, signOut, reset]
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionValue {
  const ctx = useContext(SessionContext);
  if (!ctx) throw new Error("useSession must be used within SessionProvider");
  return ctx;
}

/** Updates the cached own profile after a mutation. */
export function useSetProfile() {
  const client = useQueryClient();
  return useCallback((profile: OwnProfile) => client.setQueryData(queryKeys.profile, profile), [client]);
}
