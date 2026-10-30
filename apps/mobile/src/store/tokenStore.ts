import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "alba.auth.tokens";
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
    // Corrupted payload: treat as signed out.
  }
  return null;
}

// SecureStore (Keychain / Keystore) on devices; localStorage on web, where no
// secure storage exists. Memory is the last-resort fallback.
async function readRaw(): Promise<string | null> {
  try {
    if (Platform.OS === "web") {
      return globalThis.localStorage?.getItem(TOKENS_KEY) ?? null;
    }
    return await SecureStore.getItemAsync(TOKENS_KEY);
  } catch {
    return null;
  }
}

async function writeRaw(value: string | null): Promise<void> {
  try {
    if (Platform.OS === "web") {
      if (value === null) {
        globalThis.localStorage?.removeItem(TOKENS_KEY);
      } else {
        globalThis.localStorage?.setItem(TOKENS_KEY, value);
      }
      return;
    }
    if (value === null) {
      await SecureStore.deleteItemAsync(TOKENS_KEY);
    } else {
      await SecureStore.setItemAsync(TOKENS_KEY, value);
    }
  } catch {
    // Keep the in-memory copy when storage is unavailable.
  }
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  await writeRaw(JSON.stringify(tokens));
}

export async function loadTokens(): Promise<AuthTokens | null> {
  memoryTokens = parseTokens(await readRaw()) ?? memoryTokens;
  return memoryTokens;
}

export function currentTokens(): AuthTokens | null {
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  await writeRaw(null);
}
