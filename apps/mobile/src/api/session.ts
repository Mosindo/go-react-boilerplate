import * as SecureStore from "expo-secure-store";

export type StoredTokens = { accessToken: string; refreshToken: string };

const TOKENS_KEY = "sillage.auth.tokens";

let memory: StoredTokens | null = null;
let loaded = false;
const listeners = new Set<(tokens: StoredTokens | null) => void>();

function emit(): void {
  listeners.forEach((listener) => listener(memory));
}

/** Subscribe to token changes (login, refresh, logout, forced sign-out). */
export function subscribeToSession(listener: (tokens: StoredTokens | null) => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function isStoredTokens(value: unknown): value is StoredTokens {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const candidate = value as Record<string, unknown>;
  return typeof candidate.accessToken === "string" && typeof candidate.refreshToken === "string";
}

export async function loadTokens(): Promise<StoredTokens | null> {
  if (loaded) {
    return memory;
  }
  try {
    const raw = await SecureStore.getItemAsync(TOKENS_KEY);
    if (raw) {
      const parsed: unknown = JSON.parse(raw);
      if (isStoredTokens(parsed)) {
        memory = parsed;
      }
    }
  } catch {
    // Secure storage unavailable: the session then only lives in memory.
  }
  loaded = true;
  return memory;
}

export async function saveTokens(tokens: StoredTokens): Promise<void> {
  memory = tokens;
  loaded = true;
  try {
    await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  } catch {
    // Keep the in-memory session.
  }
  emit();
}

export async function clearTokens(): Promise<void> {
  memory = null;
  loaded = true;
  try {
    await SecureStore.deleteItemAsync(TOKENS_KEY);
  } catch {
    // Nothing to clean up.
  }
  emit();
}

export function getTokensSync(): StoredTokens | null {
  return memory;
}
