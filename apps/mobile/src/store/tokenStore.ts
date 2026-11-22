import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "lueur.auth.tokens";
let memoryTokens: AuthTokens | null = null;
let loaded = false;
const listeners = new Set<(tokens: AuthTokens | null) => void>();

// Tokens live in the OS keychain/keystore on devices. On web (used for
// development and E2E tests) they are kept in memory only: no localStorage,
// so an XSS cannot exfiltrate a long-lived refresh token.
const persistent = Platform.OS !== "web";

function emit() {
  listeners.forEach((listener) => listener(memoryTokens));
}

export function subscribeTokens(listener: (tokens: AuthTokens | null) => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export async function loadTokens(): Promise<AuthTokens | null> {
  if (loaded || !persistent) {
    loaded = true;
    return memoryTokens;
  }
  loaded = true;
  try {
    const raw = await SecureStore.getItemAsync(TOKENS_KEY);
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

export function getTokens(): AuthTokens | null {
  return memoryTokens;
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  emit();
  if (!persistent) {
    return;
  }
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Keychain unavailable: the session stays in memory for this run.
  }
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  emit();
  if (!persistent) {
    return;
  }
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // Nothing to clean.
  }
}
