import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "aurore.auth.tokens";
let memoryTokens: AuthTokens | null = null;
const listeners = new Set<(tokens: AuthTokens | null) => void>();

function notify() {
  listeners.forEach((listener) => listener(memoryTokens));
}

/** Subscribe to token changes (login, silent refresh, logout). */
export function subscribeTokens(listener: (tokens: AuthTokens | null) => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

async function writeStorage(value: string | null): Promise<void> {
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
    // Storage can be unavailable (private mode, unsupported platform): keep the in-memory session.
  }
}

async function readStorage(): Promise<string | null> {
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
  notify();
  await writeStorage(JSON.stringify(tokens));
}

export async function getTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) {
    return memoryTokens;
  }
  const raw = await readStorage();
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

export async function getAccessToken(): Promise<string | null> {
  return (await getTokens())?.accessToken ?? null;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  notify();
  await writeStorage(null);
}
