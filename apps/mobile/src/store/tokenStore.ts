import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "lueur.auth.tokens";
let memoryTokens: AuthTokens | null = null;

// Native: tokens live in the OS keychain/keystore (expo-secure-store).
// Web: sessionStorage (cleared when the tab closes) to limit exposure.
async function persist(value: string | null): Promise<void> {
  try {
    if (Platform.OS === "web") {
      if (value === null) {
        globalThis.sessionStorage?.removeItem(TOKENS_KEY);
      } else {
        globalThis.sessionStorage?.setItem(TOKENS_KEY, value);
      }
      return;
    }
    if (value === null) {
      await SecureStore.deleteItemAsync(TOKENS_KEY);
    } else {
      await SecureStore.setItemAsync(TOKENS_KEY, value);
    }
  } catch {
    // Fall back to memory only.
  }
}

async function load(): Promise<string | null> {
  try {
    if (Platform.OS === "web") {
      return globalThis.sessionStorage?.getItem(TOKENS_KEY) ?? null;
    }
    return await SecureStore.getItemAsync(TOKENS_KEY);
  } catch {
    return null;
  }
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  await persist(JSON.stringify(tokens));
}

export async function getTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) {
    return memoryTokens;
  }
  const raw = await load();
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<AuthTokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      return memoryTokens;
    }
  } catch {
    // Corrupted value: ignore.
  }
  return null;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  await persist(null);
}
