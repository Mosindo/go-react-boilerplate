import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = { accessToken: string; refreshToken: string };

const KEY = "lueur.session.v1";

// Native: Keychain / Keystore via expo-secure-store.
// Web (development & E2E only): sessionStorage, cleared when the tab closes.
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
      /* storage unavailable: keep the session in memory only */
    }
  },
  remove: () => {
    try {
      globalThis.sessionStorage?.removeItem(KEY);
    } catch {
      /* ignore */
    }
  }
};

export async function loadTokens(): Promise<AuthTokens | null> {
  try {
    const raw = Platform.OS === "web" ? webStorage.get() : await SecureStore.getItemAsync(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<AuthTokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    /* corrupted entry: treat as signed out */
  }
  return null;
}

export async function persistTokens(tokens: AuthTokens | null): Promise<void> {
  try {
    if (Platform.OS === "web") {
      if (tokens) webStorage.set(JSON.stringify(tokens));
      else webStorage.remove();
      return;
    }
    if (tokens) await SecureStore.setItemAsync(KEY, JSON.stringify(tokens));
    else await SecureStore.deleteItemAsync(KEY);
  } catch {
    /* secure storage unavailable: the in-memory session still works */
  }
}
