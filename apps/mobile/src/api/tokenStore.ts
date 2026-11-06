import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type Tokens = { accessToken: string; refreshToken: string };

const KEY = "lueur.tokens";
let memory: Tokens | null = null;

function isTokens(value: unknown): value is Tokens {
  const v = value as Partial<Tokens> | null;
  return !!v && typeof v.accessToken === "string" && typeof v.refreshToken === "string";
}

// Native uses the OS keychain/keystore. The web has no secure store: it falls back to
// localStorage (acceptable for a first-party app origin; use short access tokens).
async function readRaw(): Promise<string | null> {
  try {
    if (Platform.OS === "web") return globalThis.localStorage?.getItem(KEY) ?? null;
    return await SecureStore.getItemAsync(KEY);
  } catch {
    return null;
  }
}

async function writeRaw(value: string | null): Promise<void> {
  try {
    if (Platform.OS === "web") {
      if (value === null) globalThis.localStorage?.removeItem(KEY);
      else globalThis.localStorage?.setItem(KEY, value);
      return;
    }
    if (value === null) await SecureStore.deleteItemAsync(KEY);
    else await SecureStore.setItemAsync(KEY, value);
  } catch {
    // Storage unavailable: the in-memory copy still keeps the current session alive.
  }
}

export async function loadTokens(): Promise<Tokens | null> {
  if (memory) return memory;
  const raw = await readRaw();
  if (!raw) return null;
  try {
    const parsed: unknown = JSON.parse(raw);
    if (isTokens(parsed)) {
      memory = parsed;
      return parsed;
    }
  } catch {
    // Corrupt value: treat as signed out.
  }
  return null;
}

export function currentTokens(): Tokens | null {
  return memory;
}

export async function saveTokens(tokens: Tokens): Promise<void> {
  memory = tokens;
  await writeRaw(JSON.stringify(tokens));
}

export async function clearTokens(): Promise<void> {
  memory = null;
  await writeRaw(null);
}
