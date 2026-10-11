import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { authApi, http, isApiError } from '../../api';
import type { User } from '../../api';

type Status = 'booting' | 'signedOut' | 'signedIn' | 'bootError';

interface AuthContextValue {
  status: Status;
  user: User | null;
  /** Called after login / register succeeded (tokens are already stored). */
  signIn: (user: User) => void;
  /** Revokes the session on the server (best effort) and clears local state. */
  logout: () => Promise<void>;
  /** Drops local session state without calling the server (after account deletion). */
  endSession: () => Promise<void>;
  /** Re-reads /me (e.g. after onboarding finished or photos changed). */
  refreshUser: () => Promise<User | null>;
  retryBoot: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<Status>('booting');
  const [user, setUser] = useState<User | null>(null);
  const [bootAttempt, setBootAttempt] = useState(0);

  const dropSession = useCallback(() => {
    setUser(null);
    setStatus('signedOut');
    queryClient.clear();
  }, [queryClient]);

  // Restore a persisted session on launch.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      setStatus('booting');
      const hasSession = await http.restoreSession();
      if (!hasSession) {
        if (!cancelled) setStatus('signedOut');
        return;
      }
      try {
        const me = await authApi.me();
        if (cancelled) return;
        setUser(me);
        setStatus('signedIn');
      } catch (err) {
        if (cancelled) return;
        if (isApiError(err) && err.isNetwork) {
          setStatus('bootError');
        } else {
          await http.clearSession();
          setStatus('signedOut');
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [bootAttempt]);

  // A failed token refresh ends the session everywhere.
  useEffect(() => http.onSessionExpired(dropSession), [dropSession]);

  const signIn = useCallback((next: User) => {
    setUser(next);
    setStatus('signedIn');
  }, []);

  const logout = useCallback(async () => {
    await authApi.logout();
    dropSession();
  }, [dropSession]);

  const endSession = useCallback(async () => {
    await http.clearSession();
    dropSession();
  }, [dropSession]);

  const refreshUser = useCallback(async () => {
    try {
      const me = await authApi.me();
      setUser(me);
      return me;
    } catch {
      return null;
    }
  }, []);

  const retryBoot = useCallback(() => setBootAttempt((n) => n + 1), []);

  const value = useMemo<AuthContextValue>(
    () => ({ status, user, signIn, logout, endSession, refreshUser, retryBoot }),
    [status, user, signIn, logout, endSession, refreshUser, retryBoot],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>');
  return ctx;
}

/** The signed-in user; only call from screens rendered after authentication. */
export function useCurrentUser(): User {
  const { user } = useAuth();
  if (!user) throw new Error('No authenticated user');
  return user;
}
