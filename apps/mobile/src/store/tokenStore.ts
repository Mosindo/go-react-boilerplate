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
    // Secure storage can be unavailable (web, some emulators): the session then lives in memory only.
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
      if (parsed.accessToken && parsed.refreshToken) {
        memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
        return memoryTokens;
      }
    }
  } catch {
    // Corrupted or unavailable storage: behave as signed out.
  }
  return null;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // Nothing to clean up on platforms without secure storage.
  }
}
