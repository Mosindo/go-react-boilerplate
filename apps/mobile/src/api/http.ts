import { ApiError, errorFromResponse, networkError } from './errors';
import type { AuthResponse } from './types';

export interface Tokens {
  accessToken: string;
  refreshToken: string;
}

/** Persistence abstraction (SecureStore on native, localStorage/memory on web). */
export interface TokenStorage {
  load(): Promise<Tokens | null>;
  save(tokens: Tokens): Promise<void>;
  clear(): Promise<void>;
}

export type FetchLike = (input: string, init?: RequestInit) => Promise<Response>;

export interface HttpClientOptions {
  baseUrl: string;
  storage: TokenStorage;
  fetchImpl?: FetchLike;
  timeoutMs?: number;
  uploadTimeoutMs?: number;
}

export type QueryValue = string | number | boolean | null | undefined;

export interface RequestOptions {
  query?: Record<string, QueryValue>;
  /** JSON body. */
  body?: unknown;
  /** Multipart body (photo upload). Content-Type is left to the runtime. */
  form?: FormData;
  /**
   * `false` for public endpoints (login, register, ...). `'optional'` attaches the access token
   * when there is one but never triggers a refresh (logout). Default: required, with refresh + retry.
   */
  auth?: boolean | 'optional';
}

export interface HttpClient {
  readonly baseUrl: string;
  request<T>(method: string, path: string, options?: RequestOptions): Promise<T>;
  /** Loads persisted tokens; returns true when a session may exist. */
  restoreSession(): Promise<boolean>;
  getRefreshToken(): string | null;
  setSession(tokens: Tokens): Promise<void>;
  clearSession(): Promise<void>;
  /** Called when a refresh fails definitively (session over). */
  onSessionExpired(handler: () => void): () => void;
}

function buildQuery(query?: Record<string, QueryValue>): string {
  if (!query) return '';
  const parts: string[] = [];
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === '') continue;
    parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`);
  }
  return parts.length > 0 ? `?${parts.join('&')}` : '';
}

async function readBody(res: Response): Promise<unknown> {
  if (res.status === 204) return undefined;
  const text = await res.text();
  if (!text) return undefined;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return undefined;
  }
}

export function createHttpClient(options: HttpClientOptions): HttpClient {
  const { baseUrl, storage } = options;
  const fetchImpl: FetchLike = options.fetchImpl ?? ((input, init) => fetch(input, init));
  const timeoutMs = options.timeoutMs ?? 20_000;
  const uploadTimeoutMs = options.uploadTimeoutMs ?? 60_000;

  let tokens: Tokens | null = null;
  let restored = false;
  let refreshInFlight: Promise<Tokens> | null = null;
  const expiredHandlers = new Set<() => void>();

  async function rawFetch(
    method: string,
    path: string,
    opts: RequestOptions,
    accessToken: string | null,
  ): Promise<Response> {
    const headers: Record<string, string> = { Accept: 'application/json' };
    let body: string | FormData | undefined;
    if (opts.form) {
      body = opts.form;
    } else if (opts.body !== undefined) {
      headers['Content-Type'] = 'application/json';
      body = JSON.stringify(opts.body);
    }
    if (accessToken) headers.Authorization = `Bearer ${accessToken}`;

    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), opts.form ? uploadTimeoutMs : timeoutMs);
    try {
      return await fetchImpl(`${baseUrl}${path}${buildQuery(opts.query)}`, {
        method,
        headers,
        body,
        signal: controller.signal,
      });
    } catch {
      throw networkError(controller.signal.aborted);
    } finally {
      clearTimeout(timer);
    }
  }

  async function expireSession(): Promise<void> {
    tokens = null;
    await storage.clear().catch(() => undefined);
    expiredHandlers.forEach((handler) => handler());
  }

  /** Single-flight refresh: concurrent 401s share one /auth/refresh call. */
  function refreshTokens(): Promise<Tokens> {
    if (refreshInFlight) return refreshInFlight;
    const current = tokens;
    refreshInFlight = (async () => {
      if (!current) {
        await expireSession();
        throw new ApiError(401, 'unauthorized', 'No session');
      }
      // A network failure here propagates as-is: the session is kept for a later retry.
      const res = await rawFetch(
        'POST',
        '/auth/refresh',
        { body: { refreshToken: current.refreshToken }, auth: false },
        null,
      );
      const parsed = await readBody(res);
      if (!res.ok) {
        const apiErr = errorFromResponse(res.status, parsed);
        if (res.status >= 500 || res.status === 429) throw apiErr;
        await expireSession();
        throw apiErr;
      }
      const data = parsed as AuthResponse;
      const next: Tokens = { accessToken: data.accessToken, refreshToken: data.refreshToken };
      tokens = next;
      await storage.save(next);
      return next;
    })().finally(() => {
      refreshInFlight = null;
    });
    return refreshInFlight;
  }

  async function request<T>(method: string, path: string, opts: RequestOptions = {}): Promise<T> {
    const authed = opts.auth !== false;
    if (authed && !restored) await restoreSession();

    const usedToken = authed ? (tokens?.accessToken ?? null) : null;
    let res = await rawFetch(method, path, opts, usedToken);

    if (res.status === 401 && authed && opts.auth !== 'optional') {
      // Another request may already have rotated the token while this one was in flight.
      const latest = tokens?.accessToken ?? null;
      const fresh =
        latest && latest !== usedToken ? { accessToken: latest } : await refreshTokens();
      res = await rawFetch(method, path, opts, fresh.accessToken);
    }

    const parsed = await readBody(res);
    if (!res.ok) {
      throw errorFromResponse(res.status, parsed);
    }
    return parsed as T;
  }

  async function restoreSession(): Promise<boolean> {
    if (!restored) {
      tokens = await storage.load().catch(() => null);
      restored = true;
    }
    return tokens !== null;
  }

  return {
    baseUrl,
    request,
    restoreSession,
    getRefreshToken: () => tokens?.refreshToken ?? null,
    async setSession(next) {
      tokens = next;
      restored = true;
      await storage.save(next);
    },
    async clearSession() {
      tokens = null;
      restored = true;
      await storage.clear().catch(() => undefined);
    },
    onSessionExpired(handler) {
      expiredHandlers.add(handler);
      return () => {
        expiredHandlers.delete(handler);
      };
    },
  };
}
