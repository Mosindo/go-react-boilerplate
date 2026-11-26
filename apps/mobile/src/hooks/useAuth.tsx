import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import * as authApi from "../api/auth";
import { ApiError } from "../api/client";
import { getMyProfile } from "../api/profile";
import type { AuthResponse, AuthUser, MyProfile } from "../api/types";
import { clearTokens, currentTokens, loadTokens, saveTokens, subscribeTokens, type AuthTokens } from "../store/tokenStore";
import { queryClient, queryKeys } from "./queryClient";

export type AuthStatus = "booting" | "signedOut" | "unavailable" | "onboarding" | "ready";

type AuthContextValue = {
  status: AuthStatus;
  user: AuthUser | null;
  profile: MyProfile | null;
  signIn: (session: AuthResponse) => Promise<void>;
  signOut: () => Promise<void>;
  /** Forget the local session without calling the API (e.g. after account deletion). */
  forgetSession: () => Promise<void>;
  retry: () => void;
  /** Leaves onboarding; until then the onboarding flow stays mounted even once the profile is complete. */
  finishOnboarding: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [tokens, setTokens] = useState<AuthTokens | null>(currentTokens());
  const [restoring, setRestoring] = useState(true);
  const [onboardingActive, setOnboardingActive] = useState(false);

  useEffect(() => {
    const unsubscribe = subscribeTokens(setTokens);
    loadTokens()
      .then(setTokens)
      .finally(() => setRestoring(false));
    return unsubscribe;
  }, []);

  const signedIn = !!tokens;
  const userQuery = useQuery({ queryKey: queryKeys.me, queryFn: authApi.me, enabled: signedIn && !restoring, staleTime: Infinity });
  const profileQuery = useQuery({ queryKey: queryKeys.profile, queryFn: getMyProfile, enabled: signedIn && !restoring });

  // A definitive auth failure (refresh token rejected) ends the session.
  useEffect(() => {
    const error = userQuery.error;
    if (error instanceof ApiError && error.status === 401) {
      void clearTokens();
    }
  }, [userQuery.error]);

  const forgetSession = useCallback(async () => {
    await clearTokens();
    queryClient.clear();
  }, []);

  const signIn = useCallback(async (session: AuthResponse) => {
    queryClient.clear();
    queryClient.setQueryData(queryKeys.me, session.user);
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
  }, []);

  const signOut = useCallback(async () => {
    const refresh = currentTokens()?.refreshToken;
    if (refresh) {
      await authApi.logout(refresh).catch(() => undefined);
    }
    await forgetSession();
  }, [forgetSession]);

  const status: AuthStatus = useMemo(() => {
    if (restoring) return "booting";
    if (!signedIn) return "signedOut";
    if (!userQuery.data || !profileQuery.data) {
      return userQuery.isError || profileQuery.isError ? "unavailable" : "booting";
    }
    return profileQuery.data.completion.complete && !onboardingActive ? "ready" : "onboarding";
  }, [restoring, signedIn, userQuery.data, userQuery.isError, profileQuery.data, profileQuery.isError, onboardingActive]);

  useEffect(() => {
    if (status === "onboarding" && !onboardingActive) setOnboardingActive(true);
    if (status === "signedOut") setOnboardingActive(false);
  }, [status, onboardingActive]);

  const finishOnboarding = useCallback(() => setOnboardingActive(false), []);

  const { refetch: refetchUser } = userQuery;
  const { refetch: refetchProfile } = profileQuery;
  const retry = useCallback(() => {
    void refetchUser();
    void refetchProfile();
  }, [refetchUser, refetchProfile]);

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      user: userQuery.data ?? null,
      profile: profileQuery.data ?? null,
      signIn,
      signOut,
      forgetSession,
      retry,
      finishOnboarding
    }),
    [status, userQuery.data, profileQuery.data, signIn, signOut, forgetSession, retry, finishOnboarding]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}

/** Writes a fresh profile returned by a mutation into the cache. */
export function setProfileCache(profile: MyProfile) {
  queryClient.setQueryData(queryKeys.profile, profile);
}
