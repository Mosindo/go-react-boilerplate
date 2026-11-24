import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

// Native: Keychain / Keystore via expo-secure-store.
// Web (development & E2E only): sessionStorage, cleared with the tab.
const webStore = {
  get(key: string): string | null {
    try {
      return globalThis.sessionStorage?.getItem(key) ?? null;
    } catch {
      return null;
    }
  },
  set(key: string, value: string) {
    try {
      globalThis.sessionStorage?.setItem(key, value);
    } catch {
      // Storage may be disabled; the session then lives in memory only.
    }
  },
  remove(key: string) {
    try {
      globalThis.sessionStorage?.removeItem(key);
    } catch {
      // ignore
    }
  }
};

export async function readSecure(key: string): Promise<string | null> {
  if (Platform.OS === "web") {
    return webStore.get(key);
  }
  try {
    return await SecureStore.getItemAsync(key);
  } catch {
    return null;
  }
}

export async function writeSecure(key: string, value: string): Promise<void> {
  if (Platform.OS === "web") {
    webStore.set(key, value);
    return;
  }
  try {
    await SecureStore.setItemAsync(key, value);
  } catch {
    // Keep the in-memory session if secure storage is unavailable.
  }
}

export async function removeSecure(key: string): Promise<void> {
  if (Platform.OS === "web") {
    webStore.remove(key);
    return;
  }
  try {
    await SecureStore.deleteItemAsync(key);
  } catch {
    // ignore
  }
}
