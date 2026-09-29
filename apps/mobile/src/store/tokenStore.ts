import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "alba.auth.tokens";
let memoryTokens: AuthTokens | null = null;

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Fallback keeps the session in memory when secure storage is unavailable.
  }
}

export async function getTokens(): Promise<AuthTokens | null> {
  try {
    const serializedTokens = await SecureStore.getItemAsync(TOKENS_KEY);
    if (serializedTokens) {
      const parsedTokens = JSON.parse(serializedTokens) as Partial<AuthTokens>;
      if (parsedTokens.accessToken && typeof parsedTokens.refreshToken === "string") {
        memoryTokens = {
          accessToken: parsedTokens.accessToken,
          refreshToken: parsedTokens.refreshToken
        };
        return memoryTokens;
      }
    }
  } catch {
    // Ignore and fallback to memory.
  }
  return memoryTokens;
}

export async function getAccessToken(): Promise<string | null> {
  const tokens = await getTokens();
  return tokens?.accessToken ?? null;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // Ignore cleanup errors on unsupported platforms.
  }
}

// Backward-compatible helpers for older call sites that still expect token-only storage.
export async function saveToken(token: string): Promise<void> {
  await saveTokens({ accessToken: token, refreshToken: "" });
}

export async function getToken(): Promise<string | null> {
  return getAccessToken();
}

export async function clearToken(): Promise<void> {
  await clearTokens();
}
