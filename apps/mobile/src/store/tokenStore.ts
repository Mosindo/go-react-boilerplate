import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "lien.auth.tokens";
let memoryTokens: AuthTokens | null = null;

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Secure storage unavailable: the session lives in memory only.
  }
}

export async function getTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) {
    return memoryTokens;
  }
  try {
    const serialized = await SecureStore.getItemAsync(TOKENS_KEY);
    if (serialized) {
      const parsed = JSON.parse(serialized) as Partial<AuthTokens>;
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    }
  } catch {
    // Unreadable or corrupt entry: treat as signed out.
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
    // Nothing to clean on platforms without secure storage.
  }
}
