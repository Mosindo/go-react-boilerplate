import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import type { AuthResponse, User } from "../api/types";
import { clearTokens, getTokens, onTokensChange, restoreTokens, storeTokens } from "./session";

type AuthState = {
  booting: boolean;
  user: User | null;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears local state after the server erased the account. */
  afterAccountDeleted: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [booting, setBooting] = useState(true);
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    let active = true;
    (async () => {
      const tokens = await restoreTokens();
      if (tokens) {
        try {
          const me = await api.me();
          if (active) {
            setUser(me);
          }
        } catch {
          // Invalid or expired session: the client already cleared tokens on 401.
        }
      }
      if (active) {
        setBooting(false);
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  // A failed refresh clears tokens from the API layer: drop the user too.
  useEffect(
    () =>
      onTokensChange((tokens) => {
        if (!tokens) {
          setUser(null);
          queryClient.clear();
        }
      }),
    [queryClient]
  );

  const adopt = useCallback(async (response: AuthResponse) => {
    await storeTokens({ accessToken: response.accessToken, refreshToken: response.refreshToken });
    setUser(response.user);
  }, []);

  const value = useMemo<AuthState>(
    () => ({
      booting,
      user,
      signIn: async (email, password) => adopt(await api.login(email.trim(), password)),
      signUp: async (email, password) => adopt(await api.register(email.trim(), password)),
      signOut: async () => {
        const tokens = getTokens();
        if (tokens) {
          await api.logout(tokens.refreshToken).catch(() => undefined);
        }
        await clearTokens();
      },
      afterAccountDeleted: () => clearTokens()
    }),
    [adopt, booting, user]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return ctx;
}
