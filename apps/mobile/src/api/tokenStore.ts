import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type Tokens = { accessToken: string; refreshToken: string };

const KEY = "lueur.auth.tokens";
let memory: Tokens | null = null;

// Native: Keychain/Keystore via SecureStore. Web: sessionStorage (cleared with the tab),
// which limits the exposure of refresh tokens on shared computers.
async function read(): Promise<string | null> {
  if (Platform.OS === "web") {
    return globalThis.sessionStorage?.getItem(KEY) ?? null;
  }
  return SecureStore.getItemAsync(KEY);
}

async function write(value: string | null): Promise<void> {
  if (Platform.OS === "web") {
    if (value === null) globalThis.sessionStorage?.removeItem(KEY);
    else globalThis.sessionStorage?.setItem(KEY, value);
    return;
  }
  if (value === null) await SecureStore.deleteItemAsync(KEY);
  else await SecureStore.setItemAsync(KEY, value);
}

export async function loadTokens(): Promise<Tokens | null> {
  if (memory) return memory;
  try {
    const raw = await read();
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<Tokens>;
    if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
      memory = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
      return memory;
    }
  } catch {
    // Corrupted storage: treat as signed out.
  }
  return null;
}

export async function saveTokens(tokens: Tokens): Promise<void> {
  memory = tokens;
  try {
    await write(JSON.stringify(tokens));
  } catch {
    // Keep the in-memory session if persistent storage is unavailable.
  }
}

export async function clearTokens(): Promise<void> {
  memory = null;
  try {
    await write(null);
  } catch {
    // Ignore.
  }
}
