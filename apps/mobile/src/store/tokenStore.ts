import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "lueur.auth.tokens";
let memoryTokens: AuthTokens | null = null;

// Tokens live in the platform keychain/keystore. On web, SecureStore is not
// available: tokens are kept in memory only (a reload signs the user out),
// which avoids exposing refresh tokens to persistent browser storage.
const secureStoreAvailable = Platform.OS !== "web";

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  if (!secureStoreAvailable) {
    return;
  }
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Keep the in-memory session if secure storage fails.
  }
}

export async function loadTokens(): Promise<AuthTokens | null> {
  if (memoryTokens || !secureStoreAvailable) {
    return memoryTokens;
  }
  try {
    const raw = await SecureStore.getItemAsync(TOKENS_KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as Partial<AuthTokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      memoryTokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    memoryTokens = null;
  }
  return memoryTokens;
}

export function currentTokens(): AuthTokens | null {
  return memoryTokens;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  if (!secureStoreAvailable) {
    return;
  }
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // Nothing else to clean up.
  }
}
