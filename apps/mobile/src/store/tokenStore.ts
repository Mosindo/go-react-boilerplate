import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "alma.auth.tokens";
let memoryTokens: AuthTokens | null = null;

function isTokens(value: unknown): value is AuthTokens {
  const candidate = value as Partial<AuthTokens> | null;
  return typeof candidate?.accessToken === "string" && typeof candidate?.refreshToken === "string";
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Secure storage can be unavailable (web, some emulators): keep the session in memory only.
  }
}

/** Reads tokens from memory first and falls back to secure storage after a cold start. */
export async function getTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) {
    return memoryTokens;
  }
  try {
    const serialized = await SecureStore.getItemAsync(TOKENS_KEY);
    if (serialized) {
      const parsed: unknown = JSON.parse(serialized);
      if (isTokens(parsed)) {
        memoryTokens = parsed;
      }
    }
  } catch {
    // Corrupted or unavailable storage: behave as signed out.
  }
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // Nothing to clean on platforms without secure storage.
  }
}
