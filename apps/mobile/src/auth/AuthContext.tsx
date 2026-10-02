import { useQueryClient } from "@tanstack/react-query";
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { authApi } from "../api/endpoints";
import { configureSession, type Tokens } from "../api/client";
import type { AuthResponse, User } from "../api/types";
import { getItem, removeItem, setItem } from "../lib/storage";

const STORAGE_KEY = "alba.session.v1";

type Status = "booting" | "anonymous" | "authenticated";

interface AuthState {
  status: Status;
  user: User | null;
}

export interface AuthApi extends AuthState {
  register: (email: string, password: string) => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  /** Clears the local session after the server already ended it (account deleted). */
  forgetSession: () => Promise<void>;
}

const AuthContext = createContext<AuthApi | null>(null);

export function useAuth(): AuthApi {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}

interface Persisted extends Tokens {
  user: User;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [state, setState] = useState<AuthState>({ status: "booting", user: null });
  const tokensRef = useRef<Tokens | null>(null);
  const userRef = useRef<User | null>(null);

  const clear = useCallback(async () => {
    tokensRef.current = null;
    userRef.current = null;
    await removeItem(STORAGE_KEY);
    queryClient.clear(); // never leak the previous user's cache to the next session
    setState({ status: "anonymous", user: null });
  }, [queryClient]);

  const persist = useCallback(async (tokens: Tokens, user: User) => {
    tokensRef.current = tokens;
    userRef.current = user;
    const value: Persisted = { ...tokens, user };
    await setItem(STORAGE_KEY, JSON.stringify(value));
  }, []);

  useEffect(() => {
    configureSession({
      getTokens: () => tokensRef.current,
      setTokens: (tokens) => {
        tokensRef.current = tokens;
        if (userRef.current) void persist(tokens, userRef.current);
      },
      onAuthLost: () => void clear(),
    });
    return () => configureSession(null);
  }, [clear, persist]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const raw = await getItem(STORAGE_KEY);
      if (!raw) {
        if (!cancelled) setState({ status: "anonymous", user: null });
        return;
      }
      try {
        const saved = JSON.parse(raw) as Persisted;
        tokensRef.current = { accessToken: saved.accessToken, refreshToken: saved.refreshToken };
        userRef.current = saved.user;
        const user = await authApi.me(); // refreshes the token transparently when expired
        if (cancelled) return;
        await persist(tokensRef.current ?? saved, user);
        setState({ status: "authenticated", user });
      } catch (err) {
        if (cancelled) return;
        // Offline at start: keep the saved session instead of logging the person out.
        const offline =
          err instanceof Error && "isNetwork" in err && (err as { isNetwork: boolean }).isNetwork;
        if (offline && userRef.current)
          setState({ status: "authenticated", user: userRef.current });
        else await clear();
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [clear, persist]);

  const accept = useCallback(
    async (res: AuthResponse) => {
      queryClient.clear();
      await persist({ accessToken: res.accessToken, refreshToken: res.refreshToken }, res.user);
      setState({ status: "authenticated", user: res.user });
    },
    [persist, queryClient],
  );

  const api = useMemo<AuthApi>(
    () => ({
      ...state,
      register: async (email, password) => accept(await authApi.register(email, password)),
      login: async (email, password) => accept(await authApi.login(email, password)),
      logout: async () => {
        const refresh = tokensRef.current?.refreshToken;
        if (refresh) await authApi.logout(refresh).catch(() => undefined);
        await clear();
      },
      forgetSession: clear,
    }),
    [state, accept, clear],
  );

  return <AuthContext.Provider value={api}>{children}</AuthContext.Provider>;
}
