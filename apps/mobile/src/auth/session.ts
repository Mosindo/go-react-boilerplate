import { clearTokens, saveTokens, type Tokens } from "./tokenStore";

type Listener = (signedIn: boolean) => void;

let current: Tokens | null = null;
let refreshing: Promise<Tokens | null> | null = null;
let refreshImpl: ((refreshToken: string) => Promise<Tokens>) | null = null;
const listeners = new Set<Listener>();

/** In-memory session shared by the API client and the realtime socket. */
export const session = {
  get tokens(): Tokens | null {
    return current;
  },

  async set(tokens: Tokens): Promise<void> {
    current = tokens;
    await saveTokens(tokens);
    listeners.forEach((l) => l(true));
  },

  /** Restore tokens read from storage without notifying (used at boot). */
  restore(tokens: Tokens | null): void {
    current = tokens;
  },

  async clear(): Promise<void> {
    current = null;
    refreshing = null;
    await clearTokens();
    listeners.forEach((l) => l(false));
  },

  subscribe(listener: Listener): () => void {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },

  /** Registered by the API layer to avoid a circular import. */
  setRefreshImpl(fn: (refreshToken: string) => Promise<Tokens>): void {
    refreshImpl = fn;
  },

  /**
   * Exchanges the refresh token for a new pair. Concurrent callers share one
   * request: refresh tokens rotate, so a second parallel call would fail.
   */
  refresh(): Promise<Tokens | null> {
    if (!current || !refreshImpl) return Promise.resolve(null);
    if (refreshing) return refreshing;
    const impl = refreshImpl;
    const used = current.refreshToken;
    refreshing = (async () => {
      try {
        const next = await impl(used);
        await session.set(next);
        return next;
      } catch (err) {
        const status = (err as { status?: number }).status;
        // Only a definitive rejection ends the session; network errors keep it.
        if (status === 401 || status === 403) await session.clear();
        return null;
      } finally {
        refreshing = null;
      }
    })();
    return refreshing;
  }
};

/** Seconds until the access token expires (0 when unreadable). */
export function accessTokenTtl(token: string): number {
  try {
    const payload = token.split(".")[1];
    if (!payload) return 0;
    const json = decodeBase64Url(payload);
    const exp = (JSON.parse(json) as { exp?: number }).exp;
    return exp ? exp - Math.floor(Date.now() / 1000) : 0;
  } catch {
    return 0;
  }
}

function decodeBase64Url(input: string): string {
  const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
  const clean = input.replace(/-/g, "+").replace(/_/g, "/");
  let bits = 0;
  let value = 0;
  let out = "";
  for (const ch of clean) {
    const idx = chars.indexOf(ch);
    if (idx < 0) continue;
    value = (value << 6) | idx;
    bits += 6;
    if (bits >= 8) {
      bits -= 8;
      out += String.fromCharCode((value >> bits) & 0xff);
    }
  }
  return out;
}

/** Returns a token valid for at least 30 more seconds, refreshing if needed. */
export async function getFreshAccessToken(): Promise<string | null> {
  if (!current) return null;
  if (accessTokenTtl(current.accessToken) > 30) return current.accessToken;
  const next = await session.refresh();
  return next?.accessToken ?? null;
}
