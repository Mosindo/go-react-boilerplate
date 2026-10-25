import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { ApiError, authApi, onSessionExpired, tokenStore, type AuthResponse, type Me } from "../../api";
import { queryClient } from "../../queryClient";
import { showToast } from "../../shared/feedback";

type AuthStatus = "booting" | "signedOut" | "signedIn";

type AuthContextValue = {
  status: AuthStatus;
  user: Me | null;
  /** True when the stored session could not be verified because the server is unreachable. */
  bootFailed: boolean;
  retryBoot: () => void;
  signIn: (response: AuthResponse) => Promise<void>;
  /** Revokes the refresh token server-side (best effort) and clears local state. */
  signOut: () => Promise<void>;
  /** Clears local state only (used after the account has been deleted). */
  endSession: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("booting");
  const [user, setUser] = useState<Me | null>(null);
  const [bootFailed, setBootFailed] = useState(false);
  const [bootAttempt, setBootAttempt] = useState(0);

  useEffect(() => {
    let active = true;
    async function boot() {
      setBootFailed(false);
      const stored = await tokenStore.load();
      if (!stored) {
        if (active) setStatus("signedOut");
        return;
      }
      try {
        const me = await authApi.me();
        if (active) {
          setUser(me);
          setStatus("signedIn");
        }
      } catch (error) {
        if (!active) return;
        if (error instanceof ApiError && error.status !== 0 && error.status < 500) {
          await tokenStore.clear();
          setStatus("signedOut");
        } else {
          setBootFailed(true);
        }
      }
    }
    void boot();
    return () => {
      active = false;
    };
  }, [bootAttempt]);

  const endSession = useCallback(async () => {
    await tokenStore.clear();
    queryClient.clear();
    setUser(null);
    setStatus("signedOut");
  }, []);

  useEffect(
    () =>
      onSessionExpired(() => {
        queryClient.clear();
        setUser(null);
        setStatus("signedOut");
        showToast("Votre session a expiré. Reconnectez-vous.", "error");
      }),
    []
  );

  const signIn = useCallback(async (response: AuthResponse) => {
    await tokenStore.set({ accessToken: response.accessToken, refreshToken: response.refreshToken });
    queryClient.clear();
    setUser(response.user);
    setStatus("signedIn");
  }, []);

  const signOut = useCallback(async () => {
    const refreshToken = tokenStore.get()?.refreshToken;
    if (refreshToken) {
      try {
        await authApi.logout(refreshToken);
      } catch {
        // Best effort: the local session is cleared regardless.
      }
    }
    await endSession();
  }, [endSession]);

  const retryBoot = useCallback(() => setBootAttempt((value) => value + 1), []);

  const value = useMemo<AuthContextValue>(
    () => ({ status, user, bootFailed, retryBoot, signIn, signOut, endSession }),
    [status, user, bootFailed, retryBoot, signIn, signOut, endSession]
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

/** The signed-in account. Only call from screens rendered after authentication. */
export function useCurrentUser(): Me {
  const { user } = useAuth();
  if (!user) {
    throw new Error("useCurrentUser requires a signed-in user");
  }
  return user;
}
