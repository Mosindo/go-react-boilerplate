import * as SecureStore from "expo-secure-store";
import { Platform } from "react-native";

import type { AuthUser } from "../api/types";

export type StoredTokens = {
  accessToken: string;
  refreshToken: string;
  /** Last known account, used to start offline. Not sensitive (id, e-mail). */
  user?: AuthUser;
};

const KEY = "seren.session.v1";

// Native builds keep tokens in the OS keychain/keystore. The web build (used
// for previews and automated UI tests) falls back to localStorage.
const webStorage = {
  get: () => {
    try {
      return globalThis.localStorage?.getItem(KEY) ?? null;
    } catch {
      return null;
    }
  },
  set: (v: string) => {
    try {
      globalThis.localStorage?.setItem(KEY, v);
    } catch {
      // Storage unavailable (private mode): session stays in memory only.
    }
  },
  remove: () => {
    try {
      globalThis.localStorage?.removeItem(KEY);
    } catch {
      // ignore
    }
  }
};

export async function loadTokens(): Promise<StoredTokens | null> {
  try {
    const raw = Platform.OS === "web" ? webStorage.get() : await SecureStore.getItemAsync(KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as Partial<StoredTokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken, user: parsed.user };
    }
  } catch {
    // Corrupted entry: treat as signed out.
  }
  return null;
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
    // Keychain unavailable: the in-memory session keeps working.
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
