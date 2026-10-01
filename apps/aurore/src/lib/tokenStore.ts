import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type StoredTokens = { accessToken: string; refreshToken: string };

const KEY = "aurore.session";

// Native: encrypted keychain/keystore. Web: localStorage (SecureStore is unavailable there).
async function read(): Promise<string | null> {
  try {
    if (Platform.OS === "web") return globalThis.localStorage?.getItem(KEY) ?? null;
    return await SecureStore.getItemAsync(KEY);
  } catch {
    return null;
  }
}

async function write(value: string | null): Promise<void> {
  try {
    if (Platform.OS === "web") {
      if (value === null) globalThis.localStorage?.removeItem(KEY);
      else globalThis.localStorage?.setItem(KEY, value);
      return;
    }
    if (value === null) await SecureStore.deleteItemAsync(KEY);
    else await SecureStore.setItemAsync(KEY, value);
  } catch {
    // Storage failure only costs the user a re-login; never crash the app for it.
  }
}

let cache: StoredTokens | null | undefined;

export async function loadTokens(): Promise<StoredTokens | null> {
  if (cache !== undefined) return cache;
  const raw = await read();
  if (!raw) {
    cache = null;
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<StoredTokens>;
    cache =
      typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string"
        ? { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken }
        : null;
  } catch {
    cache = null;
  }
  return cache;
}

export async function saveTokens(tokens: StoredTokens): Promise<void> {
  cache = tokens;
  await write(JSON.stringify(tokens));
}

export async function clearTokens(): Promise<void> {
  cache = null;
  await write(null);
}

export function resetTokenCacheForTests(): void {
  cache = undefined;
}
