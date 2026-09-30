import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type Tokens = { accessToken: string; refreshToken: string };

const STORAGE_KEY = "lueur.session.v1";
let current: Tokens | null = null;
const listeners = new Set<(tokens: Tokens | null) => void>();

// Native: Keychain/Keystore via SecureStore. Web: sessionStorage (cleared when
// the tab closes; never localStorage, to limit exposure).
async function persist(tokens: Tokens | null): Promise<void> {
  try {
    if (Platform.OS === "web") {
      if (tokens) {
        globalThis.sessionStorage?.setItem(STORAGE_KEY, JSON.stringify(tokens));
      } else {
        globalThis.sessionStorage?.removeItem(STORAGE_KEY);
      }
      return;
    }
    if (tokens) {
      await SecureStore.setItemAsync(STORAGE_KEY, JSON.stringify(tokens));
    } else {
      await SecureStore.deleteItemAsync(STORAGE_KEY);
    }
  } catch {
    // Storage unavailable: the session simply stays in memory.
  }
}

async function read(): Promise<Tokens | null> {
  try {
    const raw =
      Platform.OS === "web" ? globalThis.sessionStorage?.getItem(STORAGE_KEY) ?? null : await SecureStore.getItemAsync(STORAGE_KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as Partial<Tokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    // Corrupted entry: treat as signed out.
  }
  return null;
}

export const session = {
  get(): Tokens | null {
    return current;
  },
  async restore(): Promise<Tokens | null> {
    current = await read();
    return current;
  },
  async set(tokens: Tokens | null): Promise<void> {
    current = tokens;
    listeners.forEach((listener) => listener(tokens));
    await persist(tokens);
  },
  subscribe(listener: (tokens: Tokens | null) => void): () => void {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }
};
