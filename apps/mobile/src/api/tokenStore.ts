import * as SecureStore from "expo-secure-store";

export type AuthTokens = { accessToken: string; refreshToken: string };

const TOKENS_KEY = "lueur.session";
let memoryTokens: AuthTokens | null = null;
let loaded = false;

// Tokens live in the OS keychain/keystore. Where SecureStore is unavailable
// (web preview), they are kept in memory only and never in localStorage.
export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  loaded = true;
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Memory fallback.
  }
}

export async function getTokens(): Promise<AuthTokens | null> {
  if (loaded) {
    return memoryTokens;
  }
  try {
    const raw = await SecureStore.getItemAsync(TOKENS_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<AuthTokens>;
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    }
  } catch {
    // Memory fallback.
  }
  loaded = true;
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  loaded = true;
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // Nothing persisted.
  }
}
