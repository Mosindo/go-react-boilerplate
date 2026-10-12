import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "app.auth.tokens";
let memoryTokens: AuthTokens | null = null;

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Secure storage can be unavailable (web, simulators): keep the session in memory.
  }
}

export async function getTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) {
    return memoryTokens;
  }
  try {
    const raw = await SecureStore.getItemAsync(TOKENS_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<AuthTokens>;
      if (parsed.accessToken && parsed.refreshToken) {
        memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    }
  } catch {
    // fall through to memory
  }
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // nothing to clean on unsupported platforms
  }
}
