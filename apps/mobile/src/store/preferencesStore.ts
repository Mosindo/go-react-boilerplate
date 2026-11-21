import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

// Small non-sensitive UI preferences (theme). SecureStore on native,
// localStorage on web.
const PREFIX = "lueur.pref.";

export async function readPreference(key: string): Promise<string | null> {
  try {
    if (Platform.OS === "web") {
      return globalThis.localStorage?.getItem(PREFIX + key) ?? null;
    }
    return await SecureStore.getItemAsync(PREFIX + key);
  } catch {
    return null;
  }
}

export async function writePreference(key: string, value: string): Promise<void> {
  try {
    if (Platform.OS === "web") {
      globalThis.localStorage?.setItem(PREFIX + key, value);
      return;
    }
    await SecureStore.setItemAsync(PREFIX + key, value);
  } catch {
    // Preferences are best effort.
  }
}
