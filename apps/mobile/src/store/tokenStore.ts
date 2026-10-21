import * as SecureStore from "expo-secure-store";
import { Platform } from "react-native";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "alma.auth.tokens";
let memoryTokens: AuthTokens | null = null;

// SecureStore is native-only; on the web the session lives in localStorage (best effort).
async function readRaw(): Promise<string | null> {
  if (Platform.OS === "web") {
    try {
      return globalThis.localStorage?.getItem(TOKENS_KEY) ?? null;
    } catch {
      return null;
    }
  }
  return SecureStore.getItemAsync(TOKENS_KEY);
}

async function writeRaw(value: string | null): Promise<void> {
  if (Platform.OS === "web") {
    try {
      if (value === null) globalThis.localStorage?.removeItem(TOKENS_KEY);
      else globalThis.localStorage?.setItem(TOKENS_KEY, value);
    } catch {
      // storage unavailable: keep the in-memory session only
    }
    return;
  }
  if (value === null) await SecureStore.deleteItemAsync(TOKENS_KEY);
  else await SecureStore.setItemAsync(TOKENS_KEY, value);
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  try {
    await writeRaw(JSON.stringify(tokens));
  } catch {
    // secure storage unavailable: keep the session in memory
  }
}

export async function getTokens(): Promise<AuthTokens | null> {
  if (memoryTokens) return memoryTokens;
  try {
    const raw = await readRaw();
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<AuthTokens>;
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    }
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
    // nothing to clean
  }
}
