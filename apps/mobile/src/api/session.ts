import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type StoredTokens = {
  accessToken: string;
  refreshToken: string;
};

const KEY = "lueur.session.v1";
let memory: StoredTokens | null = null;

// Native: tokens live in the Keychain / Keystore via SecureStore.
// Web (development/testing target): sessionStorage, cleared with the tab.
const webStore = {
  get(): string | null {
    try {
      return globalThis.sessionStorage?.getItem(KEY) ?? null;
    } catch {
      return null;
    }
  },
  set(value: string | null) {
    try {
      if (value === null) {
        globalThis.sessionStorage?.removeItem(KEY);
      } else {
        globalThis.sessionStorage?.setItem(KEY, value);
      }
    } catch {
      // storage unavailable: keep memory only
    }
  }
};

function parse(raw: string | null): StoredTokens | null {
  if (!raw) {
    return null;
  }
  try {
    const value = JSON.parse(raw) as Partial<StoredTokens>;
    if (typeof value.accessToken === "string" && typeof value.refreshToken === "string") {
      return { accessToken: value.accessToken, refreshToken: value.refreshToken };
    }
  } catch {
    // corrupted entry
  }
  return null;
}

export async function loadTokens(): Promise<StoredTokens | null> {
  if (memory) {
    return memory;
  }
  const raw = Platform.OS === "web" ? webStore.get() : await SecureStore.getItemAsync(KEY).catch(() => null);
  memory = parse(raw);
  return memory;
}

export function currentTokens(): StoredTokens | null {
  return memory;
}

export async function saveTokens(tokens: StoredTokens): Promise<void> {
  memory = tokens;
  const raw = JSON.stringify(tokens);
  if (Platform.OS === "web") {
    webStore.set(raw);
    return;
  }
  await SecureStore.setItemAsync(KEY, raw).catch(() => undefined);
}

export async function clearTokens(): Promise<void> {
  memory = null;
  if (Platform.OS === "web") {
    webStore.set(null);
    return;
  }
  await SecureStore.deleteItemAsync(KEY).catch(() => undefined);
}
