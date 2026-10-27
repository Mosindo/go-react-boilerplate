import * as SecureStore from "expo-secure-store";

export type AuthTokens = { accessToken: string; refreshToken: string };

const KEY = "ember.auth.tokens";
let memory: AuthTokens | null = null;

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memory = tokens;
  try {
    await SecureStore.setItemAsync(KEY, JSON.stringify(tokens));
  } catch {
    // Secure storage unavailable (web/simulator): the session lives in memory only.
  }
}

export async function loadTokens(): Promise<AuthTokens | null> {
  if (memory) return memory;
  try {
    const raw = await SecureStore.getItemAsync(KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<AuthTokens>;
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        memory = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    }
  } catch {
    memory = null;
  }
  return memory;
}

export function currentTokens(): AuthTokens | null {
  return memory;
}

export async function clearTokens(): Promise<void> {
  memory = null;
  try {
    await SecureStore.deleteItemAsync(KEY);
  } catch {
    // Nothing to clean up.
  }
}
