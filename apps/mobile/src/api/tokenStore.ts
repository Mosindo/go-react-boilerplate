import * as SecureStore from "expo-secure-store";
import type { TokenStore, Tokens } from "../utils/authedFetch";

const KEY = "liane.auth.tokens";
let memory: Tokens | null = null;

function isTokens(value: unknown): value is Tokens {
  if (typeof value !== "object" || value === null) return false;
  const v = value as Record<string, unknown>;
  return typeof v.accessToken === "string" && typeof v.refreshToken === "string";
}

export const tokenStore: TokenStore = {
  async get() {
    if (memory) return memory;
    try {
      const raw = await SecureStore.getItemAsync(KEY);
      if (raw) {
        const parsed: unknown = JSON.parse(raw);
        if (isTokens(parsed)) {
          memory = parsed;
          return memory;
        }
      }
    } catch {
      // secure storage unavailable: fall back to memory only
    }
    return null;
  },
  async set(tokens) {
    memory = tokens;
    try {
      await SecureStore.setItemAsync(KEY, JSON.stringify(tokens));
    } catch {
      // keep the in-memory session
    }
  },
  async clear() {
    memory = null;
    try {
      await SecureStore.deleteItemAsync(KEY);
    } catch {
      // nothing to clean
    }
  },
};
