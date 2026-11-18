import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "lueur.auth.tokens";
let memoryTokens: AuthTokens | null = null;

// SecureStore (Keychain / Keystore) on native. On web there is no secure
// storage: tokens live in sessionStorage so they disappear with the tab.
const webStorage = {
  get: () => (typeof sessionStorage === "undefined" ? null : sessionStorage.getItem(TOKENS_KEY)),
  set: (value: string) => typeof sessionStorage !== "undefined" && sessionStorage.setItem(TOKENS_KEY, value),
  remove: () => typeof sessionStorage !== "undefined" && sessionStorage.removeItem(TOKENS_KEY)
};

function parse(raw: string | null): AuthTokens | null {
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<AuthTokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    // Corrupted entry: treat as signed out.
  }
  return null;
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  const raw = JSON.stringify(tokens);
  try {
    if (Platform.OS === "web") {
      webStorage.set(raw);
    } else {
      await SecureStore.setItemAsync(TOKENS_KEY, raw);
    }
  } catch {
    // Keep the in-memory session when persistent storage is unavailable.
  }
}

export async function loadTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) {
    return memoryTokens;
  }
  try {
    const raw = Platform.OS === "web" ? webStorage.get() : await SecureStore.getItemAsync(TOKENS_KEY);
    memoryTokens = parse(raw);
  } catch {
    memoryTokens = null;
  }
  return memoryTokens;
}

export function currentTokens(): AuthTokens | null {
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  try {
    if (Platform.OS === "web") {
      webStorage.remove();
    } else {
      await SecureStore.deleteItemAsync(TOKENS_KEY);
    }
  } catch {
    // Ignore cleanup errors.
  }
}
