import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

// Holds the current tokens in memory and persists them securely:
// - native: Keychain / Keystore through expo-secure-store;
// - web: sessionStorage (cleared when the tab closes; never localStorage).
export type Tokens = { accessToken: string; refreshToken: string };

const STORAGE_KEY = "lueur.session.v1";
let tokens: Tokens | null = null;
const listeners = new Set<(tokens: Tokens | null) => void>();

function webStorage(): Storage | null {
  try {
    return typeof window !== "undefined" && window.sessionStorage ? window.sessionStorage : null;
  } catch {
    return null;
  }
}

async function persist(value: Tokens | null): Promise<void> {
  const serialized = value ? JSON.stringify(value) : null;
  try {
    if (Platform.OS === "web") {
      const storage = webStorage();
      if (!storage) return;
      if (serialized) storage.setItem(STORAGE_KEY, serialized);
      else storage.removeItem(STORAGE_KEY);
      return;
    }
    if (serialized) await SecureStore.setItemAsync(STORAGE_KEY, serialized);
    else await SecureStore.deleteItemAsync(STORAGE_KEY);
  } catch {
    // Storage unavailable: the session simply lives in memory.
  }
}

function parse(serialized: string | null): Tokens | null {
  if (!serialized) return null;
  try {
    const parsed = JSON.parse(serialized) as Partial<Tokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    // ignore corrupted value
  }
  return null;
}

export async function restoreTokens(): Promise<Tokens | null> {
  try {
    const raw = Platform.OS === "web" ? (webStorage()?.getItem(STORAGE_KEY) ?? null) : await SecureStore.getItemAsync(STORAGE_KEY);
    tokens = parse(raw);
  } catch {
    tokens = null;
  }
  return tokens;
}

export function getTokens(): Tokens | null {
  return tokens;
}

export async function setTokens(next: Tokens | null): Promise<void> {
  tokens = next;
  listeners.forEach((l) => l(next));
  await persist(next);
}

export function onTokensChange(listener: (tokens: Tokens | null) => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}
