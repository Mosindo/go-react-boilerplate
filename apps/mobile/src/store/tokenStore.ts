import { Platform } from 'react-native';
import * as SecureStore from 'expo-secure-store';

export type AuthTokens = { accessToken: string; refreshToken: string };

export interface TokenStore {
  get(): Promise<AuthTokens | null>;
  set(tokens: AuthTokens): Promise<void>;
  clear(): Promise<void>;
  /** Synchronous in-memory snapshot (null before first load). */
  peek(): AuthTokens | null;
  subscribe(listener: () => void): () => void;
}

const TOKENS_KEY = 'lumo.auth.tokens';

type Backend = {
  read(): Promise<string | null>;
  write(value: string): Promise<void>;
  remove(): Promise<void>;
};

/** SecureStore on iOS/Android, localStorage on web (documented trade-off), memory as last resort. */
function defaultBackend(): Backend {
  if (Platform.OS === 'web') {
    return {
      read: async () => globalThis.localStorage?.getItem(TOKENS_KEY) ?? null,
      write: async (v) => globalThis.localStorage?.setItem(TOKENS_KEY, v),
      remove: async () => globalThis.localStorage?.removeItem(TOKENS_KEY),
    };
  }
  return {
    read: () => SecureStore.getItemAsync(TOKENS_KEY),
    write: (v) => SecureStore.setItemAsync(TOKENS_KEY, v),
    remove: () => SecureStore.deleteItemAsync(TOKENS_KEY),
  };
}

function parseTokens(raw: string | null): AuthTokens | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<AuthTokens>;
    if (typeof parsed.accessToken === 'string' && typeof parsed.refreshToken === 'string') {
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
    }
  } catch {
    // corrupted value: treat as signed out
  }
  return null;
}

export function createTokenStore(backend: Backend = defaultBackend()): TokenStore {
  let memory: AuthTokens | null = null;
  let loaded = false;
  const listeners = new Set<() => void>();
  const emit = () => listeners.forEach((l) => l());

  return {
    async get() {
      if (loaded) return memory;
      try {
        memory = parseTokens(await backend.read());
      } catch {
        memory = null;
      }
      loaded = true;
      return memory;
    },
    async set(tokens) {
      memory = tokens;
      loaded = true;
      emit();
      try {
        await backend.write(JSON.stringify(tokens));
      } catch {
        // keep the session in memory when persistent storage is unavailable
      }
    },
    async clear() {
      memory = null;
      loaded = true;
      emit();
      try {
        await backend.remove();
      } catch {
        // ignore
      }
    },
    peek: () => memory,
    subscribe(listener) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
  };
}

export const tokenStore: TokenStore = createTokenStore();
