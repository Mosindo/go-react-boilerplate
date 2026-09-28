import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { getMe, logout as logoutRequest } from "../api/auth";
import { ApiError, errorMessage } from "../api/client";
import { setSessionExpiredHandler } from "../api/http";
import { queryKeys } from "../api/queryKeys";
import type { AuthSession, Me } from "../api/types";
import { showToast } from "../shared/feedback";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import { queryClient } from "./queryClient";

export type AuthStatus = "booting" | "signedOut" | "signedIn" | "bootError";

type AuthContextValue = {
  status: AuthStatus;
  bootError: string | null;
  retryBoot: () => void;
  /** Stores a fresh session (login/register) and loads the profile. */
  signIn: (session: AuthSession) => Promise<void>;
  /** Ends the session. Best effort revoke on the server, always clears local state. */
  signOut: () => Promise<void>;
  /** Clears local state without calling the server (after account deletion or password reset). */
  dropSession: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

function fetchMe(): Promise<Me> {
  return queryClient.fetchQuery({ queryKey: queryKeys.me, queryFn: getMe, staleTime: 0 });
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [bootError, setBootError] = useState<string | null>(null);
  const [bootAttempt, setBootAttempt] = useState(0);

  const dropSession = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
    setStatus("signedOut");
  }, []);

  useEffect(() => {
    setSessionExpiredHandler(() => {
      queryClient.clear();
      setStatus("signedOut");
      showToast("Your session expired. Please sign in again.", "info");
    });
    return () => setSessionExpiredHandler(null);
  }, []);

  useEffect(() => {
    let active = true;
    setStatus("booting");
    void (async () => {
      const tokens = await getTokens();
      if (!active) {
        return;
      }
      if (!tokens) {
        setStatus("signedOut");
        return;
      }
      try {
        await fetchMe();
        if (active) {
          setBootError(null);
          setStatus("signedIn");
        }
      } catch (error) {
        if (!active) {
          return;
        }
        if (error instanceof ApiError && error.status === 401) {
          await clearTokens();
          setStatus("signedOut");
        } else {
          setBootError(errorMessage(error, "We could not reach the server."));
          setStatus("bootError");
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [bootAttempt]);

  const signIn = useCallback(async (session: AuthSession) => {
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    try {
      await fetchMe();
    } catch (error) {
      await clearTokens();
      throw error;
    }
    setStatus("signedIn");
  }, []);

  const signOut = useCallback(async () => {
    const tokens = await getTokens();
    if (tokens) {
      try {
        await logoutRequest(tokens.refreshToken);
      } catch {
        // Best effort: the local session is dropped regardless.
      }
    }
    await dropSession();
  }, [dropSession]);

  const retryBoot = useCallback(() => setBootAttempt((value) => value + 1), []);

  const value = useMemo<AuthContextValue>(
    () => ({ status, bootError, retryBoot, signIn, signOut, dropSession }),
    [status, bootError, retryBoot, signIn, signOut, dropSession]
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

/** The signed-in user's account, profile and preferences. Only valid while signed in. */
export function useMe() {
  return useQuery({ queryKey: queryKeys.me, queryFn: getMe });
}
