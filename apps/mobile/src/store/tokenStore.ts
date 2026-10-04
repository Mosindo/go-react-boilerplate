import { Platform } from "react-native";
import * as SecureStore from "expo-secure-store";

export type AuthTokens = { accessToken: string; refreshToken: string };

/** Persistent storage abstraction (SecureStore on native, localStorage on web, memory fallback). */
export type StorageAdapter = {
  get(key: string): Promise<string | null>;
  set(key: string, value: string): Promise<void>;
  remove(key: string): Promise<void>;
};

const TOKENS_KEY = "lumen.auth.tokens";

function createDefaultStorage(): StorageAdapter {
  if (Platform.OS === "web") {
    return {
      async get(key) {
        try {
          return globalThis.localStorage?.getItem(key) ?? null;
        } catch {
          return null;
        }
      },
      async set(key, value) {
        try {
          globalThis.localStorage?.setItem(key, value);
        } catch {
          /* private mode: session lives in memory only */
        }
      },
      async remove(key) {
        try {
          globalThis.localStorage?.removeItem(key);
        } catch {
          /* ignore */
        }
      }
    };
  }
  return {
    get: (key) => SecureStore.getItemAsync(key).catch(() => null),
    set: (key, value) => SecureStore.setItemAsync(key, value).catch(() => undefined),
    remove: (key) => SecureStore.deleteItemAsync(key).catch(() => undefined)
  };
}

type Listener = () => void;

/**
 * Holds the current tokens in memory (synchronous access for the HTTP client and image
 * components) and mirrors them to persistent storage.
 */
export class TokenStore {
  private tokens: AuthTokens | null = null;
  private readonly listeners = new Set<Listener>();

  constructor(private readonly storage: StorageAdapter = createDefaultStorage()) {}

  getTokens = (): AuthTokens | null => this.tokens;

  getAccessToken = (): string | null => this.tokens?.accessToken ?? null;

  subscribe = (listener: Listener): (() => void) => {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };

  async load(): Promise<AuthTokens | null> {
    try {
      const raw = await this.storage.get(TOKENS_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as Partial<AuthTokens>;
        if (typeof parsed.accessToken === "string" && typeof parsed.refreshToken === "string") {
          this.tokens = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
          this.emit();
          return this.tokens;
        }
      }
    } catch {
      /* corrupted entry: treat as signed out */
    }
    return null;
  }

  async set(tokens: AuthTokens): Promise<void> {
    this.tokens = tokens;
    this.emit();
    await this.storage.set(TOKENS_KEY, JSON.stringify(tokens));
  }

  async clear(): Promise<void> {
    this.tokens = null;
    this.emit();
    await this.storage.remove(TOKENS_KEY);
  }

  private emit() {
    this.listeners.forEach((listener) => listener());
  }
}

export const tokenStore = new TokenStore();
