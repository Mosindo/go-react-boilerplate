import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "hearth.auth.tokens";

/**
 * Tokens live in the OS keychain/keystore through SecureStore. On platforms
 * without secure storage (web) they are kept in memory only, never in
 * localStorage.
 */
export async function saveTokens(tokens: AuthTokens): Promise<void> {
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Secure storage unavailable: session stays in memory.
  }
}

export async function loadTokens(): Promise<AuthTokens | null> {
  try {
    const raw = await SecureStore.getItemAsync(TOKENS_KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as Partial<AuthTokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    // Unavailable or corrupted.
  }
  return null;
}

export async function clearTokens(): Promise<void> {
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // Nothing to clean.
  }
}
