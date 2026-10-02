import * as SecureStore from "expo-secure-store";
import { Platform } from "react-native";

/** Secure storage on devices, localStorage on web. Never throws: storage is a convenience. */
export async function getItem(key: string): Promise<string | null> {
  try {
    if (Platform.OS === "web") return globalThis.localStorage?.getItem(key) ?? null;
    return await SecureStore.getItemAsync(key);
  } catch {
    return null;
  }
}

export async function setItem(key: string, value: string): Promise<void> {
  try {
    if (Platform.OS === "web") globalThis.localStorage?.setItem(key, value);
    else await SecureStore.setItemAsync(key, value);
  } catch {
    // ignore: the session simply will not survive a restart
  }
}

export async function removeItem(key: string): Promise<void> {
  try {
    if (Platform.OS === "web") globalThis.localStorage?.removeItem(key);
    else await SecureStore.deleteItemAsync(key);
  } catch {
    // ignore
  }
}
