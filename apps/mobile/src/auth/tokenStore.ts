import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

const KEY = "alba.refreshToken";

// Only the long-lived refresh token is persisted. The short-lived access token stays in memory.
// SecureStore (Keychain / Keystore) on devices; localStorage is the only option on the web build.

export async function loadRefreshToken(): Promise<string | null> {
  try {
    if (Platform.OS === "web") return globalThis.localStorage?.getItem(KEY) ?? null;
    return await SecureStore.getItemAsync(KEY);
  } catch {
    return null;
  }
}

export async function saveRefreshToken(token: string): Promise<void> {
  try {
    if (Platform.OS === "web") globalThis.localStorage?.setItem(KEY, token);
    else await SecureStore.setItemAsync(KEY, token);
  } catch {
    // Storage unavailable: the session simply will not survive a restart.
  }
}

export async function clearRefreshToken(): Promise<void> {
  try {
    if (Platform.OS === "web") globalThis.localStorage?.removeItem(KEY);
    else await SecureStore.deleteItemAsync(KEY);
  } catch {
    // nothing to clear
  }
}
