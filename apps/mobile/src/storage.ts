import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

/** Key/value storage backed by SecureStore on devices and localStorage on web. */
export async function getItem(key: string): Promise<string | null> {
  try {
    if (Platform.OS === "web") {
      return globalThis.localStorage?.getItem(key) ?? null;
    }
    return await SecureStore.getItemAsync(key);
  } catch {
    return null;
  }
}

export async function setItem(key: string, value: string): Promise<void> {
  try {
    if (Platform.OS === "web") {
      globalThis.localStorage?.setItem(key, value);
      return;
    }
    await SecureStore.setItemAsync(key, value);
  } catch {
    // Session stays in memory when persistent storage is unavailable.
  }
}

export async function removeItem(key: string): Promise<void> {
  try {
    if (Platform.OS === "web") {
      globalThis.localStorage?.removeItem(key);
      return;
    }
    await SecureStore.deleteItemAsync(key);
  } catch {
    // Nothing to clean up.
  }
}
