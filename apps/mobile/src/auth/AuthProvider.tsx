import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
} from 'react';
import { authApi } from '../api/auth';
import { api } from '../api/http';
import type { Me } from '../api/types';
import { fr } from '../i18n/fr';
import { queryClient } from '../query/client';
import { qk } from '../query/keys';
import { toast } from '../shared/toastStore';
import { tokenStore } from '../store/tokenStore';
import { useQuery, type UseQueryResult } from '@tanstack/react-query';

type AuthContextValue = {
  booting: boolean;
  signedIn: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Wipes the local session after the account was deleted or the password reset. */
  endSession: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

function resetClientState() {
  queryClient.clear();
  toast.clear();
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [booting, setBooting] = useState(true);
  const signedIn = useSyncExternalStore(
    tokenStore.subscribe,
    () => tokenStore.peek() !== null,
    () => false,
  );

  useEffect(() => {
    let alive = true;
    void tokenStore.get().finally(() => alive && setBooting(false));
    return () => {
      alive = false;
    };
  }, []);

  useEffect(() => {
    api.setAuthLostHandler(() => {
      resetClientState();
      toast.show(fr.errors.unauthorized, { kind: 'error' });
    });
    return () => api.setAuthLostHandler(null);
  }, []);

  const signIn = useCallback(async (email: string, password: string) => {
    const res = await authApi.login(email, password);
    queryClient.clear();
    await tokenStore.set({ accessToken: res.accessToken, refreshToken: res.refreshToken });
  }, []);

  const signUp = useCallback(async (email: string, password: string) => {
    const res = await authApi.register(email, password);
    queryClient.clear();
    await tokenStore.set({ accessToken: res.accessToken, refreshToken: res.refreshToken });
  }, []);

  const endSession = useCallback(async () => {
    await tokenStore.clear();
    resetClientState();
  }, []);

  const signOut = useCallback(async () => {
    const tokens = await tokenStore.get();
    if (tokens) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // best effort: the local session is dropped regardless
      }
    }
    await endSession();
  }, [endSession]);

  const value = useMemo(
    () => ({ booting, signedIn, signIn, signUp, signOut, endSession }),
    [booting, signedIn, signIn, signUp, signOut, endSession],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}

export function useMe(enabled = true): UseQueryResult<Me> {
  return useQuery({ queryKey: qk.me, queryFn: ({ signal }) => authApi.me(signal), enabled });
}
