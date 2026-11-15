import * as SecureStore from "expo-secure-store";
import { Platform } from "react-native";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const KEY = "lueur.session.v1";
let memory: AuthTokens | null = null;
let loaded = false;

// On native, tokens live in the OS keychain/keystore. On web (dev and E2E
// only) they live in sessionStorage, scoped to the tab.
async function read(): Promise<string | null> {
  if (Platform.OS === "web") {
    try {
      return globalThis.sessionStorage?.getItem(KEY) ?? null;
    } catch {
      return null;
    }
  }
  try {
    return await SecureStore.getItemAsync(KEY);
  } catch {
    return null;
  }
}

async function write(value: string | null): Promise<void> {
  if (Platform.OS === "web") {
    try {
      if (value === null) globalThis.sessionStorage?.removeItem(KEY);
      else globalThis.sessionStorage?.setItem(KEY, value);
    } catch {
      // Storage unavailable (private mode): the session stays in memory.
    }
    return;
  }
  try {
    if (value === null) await SecureStore.deleteItemAsync(KEY);
    else await SecureStore.setItemAsync(KEY, value);
  } catch {
    // Secure storage unavailable: the session stays in memory.
  }
}

export async function loadTokens(): Promise<AuthTokens | null> {
  if (loaded) return memory;
  const raw = await read();
  loaded = true;
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<AuthTokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      memory = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    memory = null;
  }
  return memory;
}

export function currentTokens(): AuthTokens | null {
  return memory;
}

export async function saveTokens(tokens: AuthTokens): Promise<void> {
  memory = tokens;
  loaded = true;
  await write(JSON.stringify(tokens));
}

export async function clearTokens(): Promise<void> {
  memory = null;
  loaded = true;
  await write(null);
}
