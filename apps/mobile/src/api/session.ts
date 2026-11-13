import * as SecureStore from "expo-secure-store";
import { Platform } from "react-native";

export type Tokens = {
  accessToken: string;
  refreshToken: string;
};

type Listener = (tokens: Tokens | null) => void;

const STORAGE_KEY = "aimant.session.v1";
let current: Tokens | null = null;
const listeners = new Set<Listener>();

// Native: Keychain / Keystore via SecureStore. Web (dev and E2E only): the
// session lives in memory so refresh tokens are never written to
// localStorage where any injected script could read them.
const persistent = Platform.OS !== "web";

export function getTokens(): Tokens | null {
  return current;
}

export function subscribeTokens(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export async function setTokens(tokens: Tokens | null): Promise<void> {
  current = tokens;
  listeners.forEach((listener) => listener(tokens));
  if (!persistent) {
    return;
  }
  try {
    if (tokens) {
      await SecureStore.setItemAsync(STORAGE_KEY, JSON.stringify(tokens));
    } else {
      await SecureStore.deleteItemAsync(STORAGE_KEY);
    }
  } catch {
    // Secure storage unavailable: the session stays in memory only.
  }
}

export async function restoreTokens(): Promise<Tokens | null> {
  if (!persistent) {
    return current;
  }
  try {
    const raw = await SecureStore.getItemAsync(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<Tokens>;
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        current = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    }
  } catch {
    current = null;
  }
  return current;
}
