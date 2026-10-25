import * as SecureStore from "expo-secure-store";
import type { TokenPair, TokenStorage } from "./http";

const TOKENS_KEY = "lueur.auth.tokens";

let current: TokenPair | null = null;
const listeners = new Set<() => void>();

function emit(): void {
  listeners.forEach((listener) => listener());
}

function parseTokens(raw: string | null): TokenPair | null {
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<TokenPair>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    // Corrupted payload: treat as signed out.
  }
  return null;
}

/** Token storage: synchronous in-memory cache backed by the device keychain. */
export const tokenStore: TokenStorage & {
  load(): Promise<TokenPair | null>;
  subscribe(listener: () => void): () => void;
  getAccessToken(): string | null;
} = {
  async load() {
    try {
      current = parseTokens(await SecureStore.getItemAsync(TOKENS_KEY));
    } catch {
      current = null;
    }
    emit();
    return current;
  },
  get() {
    return current;
  },
  getAccessToken() {
    return current?.accessToken ?? null;
  },
  async set(tokens) {
    current = tokens;
    emit();
    try {
      await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
    } catch {
      // Secure storage unavailable: the session stays in memory only.
    }
  },
  async clear() {
    current = null;
    emit();
    try {
      await SecureStore.deleteItemAsync(TOKENS_KEY);
    } catch {
      // Nothing to clean up on platforms without secure storage.
    }
  },
  subscribe(listener) {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }
};
