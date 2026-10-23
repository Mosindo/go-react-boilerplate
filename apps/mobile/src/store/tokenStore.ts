import * as SecureStore from "expo-secure-store";
import type { TokenPair, TokenStorage } from "../api/http";

const TOKENS_KEY = "cleo.auth.tokens";
let memoryTokens: TokenPair | null = null;

export function parseStoredTokens(raw: string | null): TokenPair | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<TokenPair>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    return null;
  }
  return null;
}

/** Secure storage with an in-memory fallback (web, or when the keychain is unavailable). */
export const tokenStorage: TokenStorage = {
  async get() {
    if (memoryTokens) return memoryTokens;
    try {
      memoryTokens = parseStoredTokens(await SecureStore.getItemAsync(TOKENS_KEY));
    } catch {
      memoryTokens = null;
    }
    return memoryTokens;
  },
  async set(tokens) {
    memoryTokens = tokens;
    try {
      await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
    } catch {
      // keep the in-memory copy
    }
  },
  async clear() {
    memoryTokens = null;
    try {
      await SecureStore.deleteItemAsync(TOKENS_KEY);
    } catch {
      // nothing to clean on unsupported platforms
    }
  }
};
