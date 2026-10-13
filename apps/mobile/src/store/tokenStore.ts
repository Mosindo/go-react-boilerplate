import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "app.auth.tokens";
let memoryTokens: AuthTokens | null = null;
const listeners = new Set<(tokens: AuthTokens | null) => void>();

function notify() {
  listeners.forEach((listener) => listener(memoryTokens));
}

/** Subscribe to token changes (login, silent refresh, forced sign-out). */
export function subscribeTokens(listener: (tokens: AuthTokens | null) => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

// SecureStore is unavailable on web: fall back to localStorage there.
async function persist(value: string | null): Promise<void> {
  try {
    if (Platform.OS === "web") {
      if (value === null) {
        globalThis.localStorage?.removeItem(TOKENS_KEY);
      } else {
        globalThis.localStorage?.setItem(TOKENS_KEY, value);
      }
      return;
    }
    if (value === null) {
      await SecureStore.deleteItemAsync(TOKENS_KEY);
    } else {
      await SecureStore.setItemAsync(TOKENS_KEY, value);
    }
  } catch {
    // The session then lives in memory only.
  }
}

async function readPersisted(): Promise<string | null> {
  try {
    if (Platform.OS === "web") {
      return globalThis.localStorage?.getItem(TOKENS_KEY) ?? null;
    }
    return await SecureStore.getItemAsync(TOKENS_KEY);
  } catch {
    return null;
  }
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  await persist(JSON.stringify(tokens));
  notify();
}

export async function loadTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) {
    return memoryTokens;
  }
  const raw = await readPersisted();
  if (raw) {
    try {
      const parsed = JSON.parse(raw) as Partial<AuthTokens>;
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    } catch {
      memoryTokens = null;
    }
  }
  return memoryTokens;
}

/** Synchronous access for code that cannot await (image headers, sockets). */
export function peekTokens(): AuthTokens | null {
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  await persist(null);
  notify();
}
