import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "app.auth.tokens";
let memoryTokens: AuthTokens | null = null;

function parseTokens(raw: string | null): AuthTokens | null {
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<AuthTokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    // Corrupted entry: treat as signed out.
  }
  return null;
}

// SecureStore (Keychain/Keystore) on native; localStorage is the only option on web.
async function readRaw(): Promise<string | null> {
  if (Platform.OS === "web") {
    return typeof localStorage === "undefined" ? null : localStorage.getItem(TOKENS_KEY);
  }
  return SecureStore.getItemAsync(TOKENS_KEY);
}

async function writeRaw(value: string | null): Promise<void> {
  if (Platform.OS === "web") {
    if (typeof localStorage === "undefined") {
      return;
    }
    if (value === null) {
      localStorage.removeItem(TOKENS_KEY);
    } else {
      localStorage.setItem(TOKENS_KEY, value);
    }
    return;
  }
  if (value === null) {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } else {
    await SecureStore.setItemAsync(TOKENS_KEY, value);
  }
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  try {
    await writeRaw(JSON.stringify(tokens));
  } catch {
    // The session then lives in memory only.
  }
}

export async function getTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) {
    return memoryTokens;
  }
  try {
    memoryTokens = parseTokens(await readRaw());
  } catch {
    memoryTokens = null;
  }
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  try {
    await writeRaw(null);
  } catch {
    // Nothing else to clean up.
  }
}
