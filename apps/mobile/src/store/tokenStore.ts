import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = { accessToken: string; refreshToken: string };

const KEY = "ember.auth.tokens";
let memory: AuthTokens | null = null;
const listeners = new Set<(tokens: AuthTokens | null) => void>();

// SecureStore is native-only; the web build falls back to sessionStorage.
async function read(): Promise<string | null> {
  if (Platform.OS === "web") {
    try {
      return globalThis.sessionStorage?.getItem(KEY) ?? null;
    } catch {
      return null;
    }
  }
  return SecureStore.getItemAsync(KEY);
}

async function write(value: string | null): Promise<void> {
  if (Platform.OS === "web") {
    try {
      if (value === null) globalThis.sessionStorage?.removeItem(KEY);
      else globalThis.sessionStorage?.setItem(KEY, value);
    } catch {
      /* storage unavailable: keep in memory */
    }
    return;
  }
  if (value === null) await SecureStore.deleteItemAsync(KEY);
  else await SecureStore.setItemAsync(KEY, value);
}

function isTokens(v: unknown): v is AuthTokens {
  const t = v as Partial<AuthTokens> | null;
  return !!t && typeof t.accessToken === "string" && typeof t.refreshToken === "string";
}

export async function loadTokens(): Promise<AuthTokens | null> {
  try {
    const raw = await read();
    const parsed: unknown = raw ? JSON.parse(raw) : null;
    memory = isTokens(parsed) ? parsed : null;
  } catch {
    memory = null;
  }
  return memory;
}

export function getTokens(): AuthTokens | null {
  return memory;
}

export async function saveTokens(tokens: AuthTokens | null): Promise<void> {
  memory = tokens;
  listeners.forEach((fn) => fn(tokens));
  try {
    await write(tokens ? JSON.stringify(tokens) : null);
  } catch {
    /* secure storage unavailable: session lives in memory only */
  }
}

export function subscribeTokens(fn: (tokens: AuthTokens | null) => void): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}
