import * as SecureStore from "expo-secure-store";
import { Platform } from "react-native";

export type AuthTokens = { accessToken: string; refreshToken: string };

const TOKENS_KEY = "tandem.auth.tokens";
let memoryTokens: AuthTokens | null = null;

function isTokens(value: unknown): value is AuthTokens {
  if (typeof value !== "object" || value === null) return false;
  const v = value as Record<string, unknown>;
  return typeof v.accessToken === "string" && typeof v.refreshToken === "string";
}

// SecureStore is unavailable on web; there tokens stay in memory only, which
// avoids persisting credentials in localStorage.
const persistent = Platform.OS !== "web";

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  if (!persistent) return;
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // keep the in-memory copy
  }
}

export async function loadTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) return memoryTokens;
  if (!persistent) return null;
  try {
    const raw = await SecureStore.getItemAsync(TOKENS_KEY);
    if (raw) {
      const parsed: unknown = JSON.parse(raw);
      if (isTokens(parsed)) {
        memoryTokens = parsed;
        return parsed;
      }
    }
  } catch {
    // fall through
  }
  return null;
}

export function getCachedTokens(): AuthTokens | null {
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  if (!persistent) return;
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // ignore
  }
}
