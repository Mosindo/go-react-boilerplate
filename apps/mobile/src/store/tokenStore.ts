import * as SecureStore from "expo-secure-store";
import { Platform } from "react-native";

export type AuthTokens = {
  accessToken: string;
  refreshToken: string;
};

const TOKENS_KEY = "lueur.auth.tokens";

/** Where tokens live between launches: Keychain/Keystore on devices, localStorage on web. */
export type TokenBackend = {
  get(): Promise<string | null>;
  set(value: string): Promise<void>;
  remove(): Promise<void>;
};

const nativeBackend: TokenBackend = {
  get: () => SecureStore.getItemAsync(TOKENS_KEY),
  set: (value) => SecureStore.setItemAsync(TOKENS_KEY, value),
  remove: () => SecureStore.deleteItemAsync(TOKENS_KEY)
};

const webBackend: TokenBackend = {
  get: async () => globalThis.localStorage?.getItem(TOKENS_KEY) ?? null,
  set: async (value) => globalThis.localStorage?.setItem(TOKENS_KEY, value),
  remove: async () => globalThis.localStorage?.removeItem(TOKENS_KEY)
};

function isTokens(value: unknown): value is AuthTokens {
  const v = value as Partial<AuthTokens> | null;
  return typeof v?.accessToken === "string" && v.accessToken !== "" && typeof v.refreshToken === "string" && v.refreshToken !== "";
}

export class TokenStore {
  private memory: AuthTokens | null = null;

  constructor(private readonly backend: TokenBackend) {}

  async save(tokens: AuthTokens): Promise<void> {
    this.memory = tokens;
    try {
      await this.backend.set(JSON.stringify(tokens));
    } catch {
      // Secure storage can be unavailable (private mode, locked keychain): keep the session in memory.
    }
  }

  async load(): Promise<AuthTokens | null> {
    if (this.memory) return this.memory;
    try {
      const raw = await this.backend.get();
      if (raw) {
        const parsed: unknown = JSON.parse(raw);
        if (isTokens(parsed)) {
          this.memory = { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken };
        }
      }
    } catch {
      // Corrupted or unreadable entry: treated as signed out.
    }
    return this.memory;
  }

  async clear(): Promise<void> {
    this.memory = null;
    try {
      await this.backend.remove();
    } catch {
      // Nothing else to clean up.
    }
  }
}

export const tokenStore = new TokenStore(Platform.OS === "web" ? webBackend : nativeBackend);
