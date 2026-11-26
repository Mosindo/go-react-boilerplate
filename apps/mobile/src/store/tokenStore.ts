import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "lueur.auth.tokens";
let memoryTokens: AuthTokens | null = null;
const listeners = new Set<(tokens: AuthTokens | null) => void>();

// On the web, tokens stay in memory only: persisting a refresh token in localStorage
// would expose it to any injected script. Native platforms use the OS keychain/keystore.
const persistent = Platform.OS !== "web";

function emit() {
  listeners.forEach((listener) => listener(memoryTokens));
}

export function subscribeTokens(listener: (tokens: AuthTokens | null) => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function currentTokens(): AuthTokens | null {
  return memoryTokens;
}

export async function loadTokens(): Promise<AuthTokens | null> {
  if (!persistent) {
    return memoryTokens;
  }
  try {
    const raw = await SecureStore.getItemAsync(TOKENS_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<AuthTokens>;
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    }
  } catch {
    memoryTokens = null;
  }
  return memoryTokens;
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  emit();
  if (persistent) {
    try {
      await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
    } catch {
      // The session still works for this run when secure storage is unavailable.
    }
  }
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  emit();
  if (persistent) {
    try {
      await SecureStore.deleteItemAsync(TOKENS_KEY);
    } catch {
      // Nothing to clean up.
    }
  }
}
