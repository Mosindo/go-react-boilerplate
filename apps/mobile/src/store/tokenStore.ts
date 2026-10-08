import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "lien.auth.tokens";
let memoryTokens: AuthTokens | null = null;

// SecureStore is native-only; the web build falls back to localStorage (same-origin, no cross-site access).
async function readRaw(): Promise<string | null> {
  if (Platform.OS === "web") {
    return typeof localStorage === "undefined" ? null : localStorage.getItem(TOKENS_KEY);
  }
  return SecureStore.getItemAsync(TOKENS_KEY);
}

async function writeRaw(value: string | null): Promise<void> {
  if (Platform.OS === "web") {
    if (typeof localStorage === "undefined") return;
    if (value === null) localStorage.removeItem(TOKENS_KEY);
    else localStorage.setItem(TOKENS_KEY, value);
    return;
  }
  if (value === null) await SecureStore.deleteItemAsync(TOKENS_KEY);
  else await SecureStore.setItemAsync(TOKENS_KEY, value);
}

export function parseTokens(raw: string | null): AuthTokens | null {
  if (!raw) return null;
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

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memoryTokens = tokens;
  try {
    await writeRaw(JSON.stringify(tokens));
  } catch {
    // Secure storage unavailable: the session survives in memory only.
  }
}

export async function getTokens(): Promise<AuthTokens | null> {
  try {
    const stored = parseTokens(await readRaw());
    if (stored) memoryTokens = stored;
    return memoryTokens;
  } catch {
    return memoryTokens;
  }
}

export function getAccessToken(): string | null {
  return memoryTokens?.accessToken ?? null;
}

export function getRefreshToken(): string | null {
  return memoryTokens?.refreshToken ?? null;
}

export async function clearTokens(): Promise<void> {
  memoryTokens = null;
  try {
    await writeRaw(null);
  } catch {
    // Nothing else to clear.
  }
}
