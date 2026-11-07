import * as SecureStore from "expo-secure-store";
import { Platform } from "react-native";

export type Tokens = { accessToken: string; refreshToken: string };

const KEY = "aube.tokens.v1";

function webStorage(): Storage | null {
  try {
    return typeof window !== "undefined" ? window.localStorage : null;
  } catch {
    return null;
  }
}

export async function loadTokens(): Promise<Tokens | null> {
  try {
    const raw = Platform.OS === "web" ? webStorage()?.getItem(KEY) : await SecureStore.getItemAsync(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<Tokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    // corrupted or unavailable storage: behave as signed out
  }
  return null;
}

export async function saveTokens(tokens: Tokens): Promise<void> {
  const raw = JSON.stringify(tokens);
  if (Platform.OS === "web") webStorage()?.setItem(KEY, raw);
  else await SecureStore.setItemAsync(KEY, raw);
}

export async function clearTokens(): Promise<void> {
  if (Platform.OS === "web") webStorage()?.removeItem(KEY);
  else await SecureStore.deleteItemAsync(KEY).catch(() => undefined);
}
