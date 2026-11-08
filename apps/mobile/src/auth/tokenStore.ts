import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type Tokens = { accessToken: string; refreshToken: string };

const KEY = "ember.auth.tokens";
let memory: Tokens | null = null;

// SecureStore (Keychain / Keystore) on devices; localStorage only on web, where nothing better exists.
async function readRaw(): Promise<string | null> {
  if (Platform.OS === "web") {
    return typeof localStorage === "undefined" ? null : localStorage.getItem(KEY);
  }
  return SecureStore.getItemAsync(KEY);
}

async function writeRaw(value: string | null): Promise<void> {
  if (Platform.OS === "web") {
    if (typeof localStorage === "undefined") return;
    if (value === null) localStorage.removeItem(KEY);
    else localStorage.setItem(KEY, value);
    return;
  }
  if (value === null) await SecureStore.deleteItemAsync(KEY);
  else await SecureStore.setItemAsync(KEY, value);
}

export async function loadTokens(): Promise<Tokens | null> {
  try {
    const raw = await readRaw();
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<Tokens>;
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        memory = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    }
  } catch {
    // Unreadable storage: fall back to whatever is in memory.
  }
  return memory;
}

export function currentTokens(): Tokens | null {
  return memory;
}

export async function saveTokens(tokens: Tokens): Promise<void> {
  memory = tokens;
  try {
    await writeRaw(JSON.stringify(tokens));
  } catch {
    // Session still works for this app run.
  }
}

export async function clearTokens(): Promise<void> {
  memory = null;
  try {
    await writeRaw(null);
  } catch {
    // Nothing to clean up.
  }
}
