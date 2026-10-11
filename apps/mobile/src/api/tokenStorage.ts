import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';
import type { TokenStorage, Tokens } from './http';

const ACCESS_KEY = 'ember.accessToken';
const REFRESH_KEY = 'ember.refreshToken';

function parseTokens(access: string | null, refresh: string | null): Tokens | null {
  return access && refresh ? { accessToken: access, refreshToken: refresh } : null;
}

/** Native: expo-secure-store (Keychain / Keystore). */
const secureStorage: TokenStorage = {
  async load() {
    const [access, refresh] = await Promise.all([
      SecureStore.getItemAsync(ACCESS_KEY),
      SecureStore.getItemAsync(REFRESH_KEY),
    ]);
    return parseTokens(access, refresh);
  },
  async save(tokens) {
    await Promise.all([
      SecureStore.setItemAsync(ACCESS_KEY, tokens.accessToken),
      SecureStore.setItemAsync(REFRESH_KEY, tokens.refreshToken),
    ]);
  },
  async clear() {
    await Promise.all([
      SecureStore.deleteItemAsync(ACCESS_KEY),
      SecureStore.deleteItemAsync(REFRESH_KEY),
    ]);
  },
};

/** Web: localStorage when available, in-memory otherwise (private mode, SSR). */
function createWebStorage(): TokenStorage {
  let memory: Tokens | null = null;
  const local = (): Storage | null => {
    try {
      return typeof localStorage === 'undefined' ? null : localStorage;
    } catch {
      return null;
    }
  };
  return {
    async load() {
      const store = local();
      if (store) {
        try {
          return parseTokens(store.getItem(ACCESS_KEY), store.getItem(REFRESH_KEY));
        } catch {
          return memory;
        }
      }
      return memory;
    },
    async save(tokens) {
      memory = tokens;
      try {
        local()?.setItem(ACCESS_KEY, tokens.accessToken);
        local()?.setItem(REFRESH_KEY, tokens.refreshToken);
      } catch {
        // Storage unavailable: the in-memory copy keeps the session alive.
      }
    },
    async clear() {
      memory = null;
      try {
        local()?.removeItem(ACCESS_KEY);
        local()?.removeItem(REFRESH_KEY);
      } catch {
        // Nothing to clear.
      }
    },
  };
}

export const tokenStorage: TokenStorage =
  Platform.OS === 'web' ? createWebStorage() : secureStorage;
