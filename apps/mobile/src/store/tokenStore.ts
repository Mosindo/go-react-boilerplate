import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "lumen.auth.tokens";
let memoryTokens: AuthTokens | null = null;
let loaded = false;

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  loaded = true;
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Secure storage unavailable (e.g. web): keep the session in memory only.
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
    // Fall back to memory.
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
    // Ignore cleanup errors on unsupported platforms.
  }
}

/** Synchronous accessor for image headers; only valid after the first getTokens() call. */
export function peekAccessToken(): string | null {
  return memoryTokens?.accessToken ?? null;
}
