import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "lueur.auth.tokens";
let memoryTokens: AuthTokens | null = null;
let loaded = false;

// Native: Keychain / Keystore via SecureStore. Web (dev/E2E only): memory + sessionStorage.
const webStorage = Platform.OS === "web" && typeof sessionStorage !== "undefined" ? sessionStorage : null;

async function readPersisted(): Promise<string | null> {
  if (webStorage) {
    return webStorage.getItem(TOKENS_KEY);
  }
  return SecureStore.getItemAsync(TOKENS_KEY);
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  loaded = true;
  try {
    if (webStorage) {
      webStorage.setItem(TOKENS_KEY, JSON.stringify(tokens));
    } else {
      await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
    }
  } catch {
    // Secure storage unavailable: the session survives in memory only.
  }
}

export async function getTokens(): Promise<AuthTokens | null> {
  if (loaded) {
    return memoryTokens;
  }
  try {
    const raw = await readPersisted();
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<AuthTokens>;
      if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
        memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      }
    }
  } catch {
    memoryTokens = null;
  }
  loaded = true;
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  loaded = true;
  try {
    if (webStorage) {
      webStorage.removeItem(TOKENS_KEY);
    } else {
      await SecureStore.deleteItemAsync(TOKENS_KEY);
    }
  } catch {
    // Nothing else to clean.
  }
}
