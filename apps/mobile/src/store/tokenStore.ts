import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type StoredTokens = { accessToken: string; refreshToken: string };

const KEY = "lueur.session.v1";

/**
 * Native: tokens live in the OS keychain / keystore (expo-secure-store).
 * Web: sessionStorage (cleared when the tab closes) — browsers have no
 * equivalent of a secure enclave, so we keep the exposure window short.
 */
const webStorage = {
  get: () => {
    try {
      return globalThis.sessionStorage?.getItem(KEY) ?? null;
    } catch {
      return null;
    }
  },
  set: (v: string) => {
    try {
      globalThis.sessionStorage?.setItem(KEY, v);
    } catch {
      // Storage unavailable (private mode): session stays in memory only.
    }
  },
  remove: () => {
    try {
      globalThis.sessionStorage?.removeItem(KEY);
    } catch {
      // ignore
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
    // Corrupted value: treat as signed out.
  }
  return null;
}

export async function loadTokens(): Promise<StoredTokens | null> {
  if (Platform.OS === "web") {
    return parse(webStorage.get());
  }
  try {
    return parse(await SecureStore.getItemAsync(KEY));
  } catch {
    return null;
  }
}

export async function saveTokens(tokens: StoredTokens): Promise<void> {
  const raw = JSON.stringify(tokens);
  if (Platform.OS === "web") {
    webStorage.set(raw);
    return;
  }
  try {
    await SecureStore.setItemAsync(KEY, raw);
  } catch {
    // Keychain unavailable: the in-memory session still works until restart.
  }
}

export async function clearTokens(): Promise<void> {
  if (Platform.OS === "web") {
    webStorage.remove();
    return;
  }
  try {
    await SecureStore.deleteItemAsync(KEY);
  } catch {
    // ignore
  }
}
