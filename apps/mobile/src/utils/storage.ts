import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

// Small key/value persistence: Keychain/Keystore on devices, sessionStorage
// on web (tokens never outlive the browser tab there).
function webStore(): Storage | null {
  try {
    return typeof window !== "undefined" ? window.sessionStorage : null;
  } catch {
    return null;
  }
}

export async function readItem(key: string): Promise<string | null> {
  try {
    if (Platform.OS === "web") {
      return webStore()?.getItem(key) ?? null;
    }
    return await SecureStore.getItemAsync(key);
  } catch {
    return null;
  }
}

export async function writeItem(key: string, value: string): Promise<void> {
  try {
    if (Platform.OS === "web") {
      webStore()?.setItem(key, value);
      return;
    }
    await SecureStore.setItemAsync(key, value);
  } catch {
    // Storage can be unavailable (private mode); the session stays in memory.
  }
}

export async function removeItem(key: string): Promise<void> {
  try {
    if (Platform.OS === "web") {
      webStore()?.removeItem(key);
      return;
    }
    await SecureStore.deleteItemAsync(key);
  } catch {
    // ignore
  }
}
