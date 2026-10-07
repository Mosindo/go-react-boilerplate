import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "ember.auth.tokens";
let memoryTokens: AuthTokens | null = null;

export function isAuthTokens(value: unknown): value is AuthTokens {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const candidate = value as Record<string, unknown>;
  return (
    typeof candidate.accessToken === "string" &&
    candidate.accessToken.length > 0 &&
    typeof candidate.refreshToken === "string" &&
    candidate.refreshToken.length > 0
  );
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = { accessToken: tokens.accessToken, refreshToken: tokens.refreshToken };
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(memoryTokens));
  } catch {
    // Secure storage unavailable (e.g. web): the session stays in memory only.
  }
}

/** Memory first (cheap, hot path for every request); secure storage on cold start. */
export async function getTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) {
    return memoryTokens;
  }
  try {
    const serialized = await SecureStore.getItemAsync(TOKENS_KEY);
    if (serialized) {
      const parsed: unknown = JSON.parse(serialized);
      if (isAuthTokens(parsed)) {
        memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    }
  } catch {
    // Corrupt or unavailable storage: treat as signed out.
  }
  return memoryTokens;
}

export async function getAccessToken(): Promise<string | null> {
  return (await getTokens())?.accessToken ?? null;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // Ignore cleanup errors on unsupported platforms.
  }
}
