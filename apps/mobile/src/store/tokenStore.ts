import * as SecureStore from "expo-secure-store";
import { useSyncExternalStore } from "react";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "alma.auth.tokens";
let memoryTokens: AuthTokens | null = null;
const listeners = new Set<() => void>();

function notify(): void {
  listeners.forEach((listener) => listener());
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  notify();
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Secure storage can be unavailable (e.g. web): the session then lives in memory only.
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
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
        notify();
      }
    }
  } catch {
    // Ignore unreadable storage and fall back to memory.
  }
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  notify();
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // Nothing to clean on unsupported platforms.
  }
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** Current access token; re-renders consumers whenever it is rotated. */
export function useAccessToken(): string | null {
  return useSyncExternalStore(
    subscribe,
    () => memoryTokens?.accessToken ?? null,
    () => null
  );
}
