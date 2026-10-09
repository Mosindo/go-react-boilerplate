import { API_BASE_URL } from '../config';
import { tokenStore, type AuthTokens, type TokenStore } from '../store/tokenStore';
import type { AuthResponse } from './types';

export type ApiErrorCode =
  | 'invalid_request'
  | 'unauthorized'
  | 'forbidden'
  | 'not_found'
  | 'conflict'
  | 'rate_limited'
  | 'profile_incomplete'
  | 'internal'
  | 'invalid_code'
  | 'photo_limit'
  | 'network'
  | 'timeout'
  | (string & {});

export class ApiError extends Error {
  readonly status: number;
  readonly code: ApiErrorCode;
  readonly retryAfter: number | null;

  constructor(
    status: number,
    code: ApiErrorCode,
    message: string,
    retryAfter: number | null = null,
  ) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.retryAfter = retryAfter;
  }
}

export type QueryParams = Record<string, string | number | boolean | undefined | null>;

export type RequestOptions = {
  body?: unknown;
  form?: FormData;
  query?: QueryParams;
  /** Set to false for endpoints that must not carry (or refresh) a session: login, register... */
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

export type ApiClientOptions = {
  baseUrl: string;
  tokens: TokenStore;
  fetchImpl?: typeof fetch;
  timeoutMs?: number;
};

const STATUS_TO_CODE: Record<number, ApiErrorCode> = {
  400: 'invalid_request',
  401: 'unauthorized',
  403: 'forbidden',
  404: 'not_found',
  409: 'conflict',
  429: 'rate_limited',
};

function buildQuery(query?: QueryParams): string {
  if (!query) return '';
  const parts: string[] = [];
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === '') continue;
    parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`);
  }
  return parts.length ? `?${parts.join('&')}` : '';
}

async function toApiError(res: Response): Promise<ApiError> {
  let payload: { error?: string; code?: string } | null;
  try {
    const text = await res.text();
    payload = text ? (JSON.parse(text) as { error?: string; code?: string }) : null;
  } catch {
    payload = null;
  }
  const retry = Number(res.headers.get('Retry-After'));
  return new ApiError(
    res.status,
    payload?.code ?? STATUS_TO_CODE[res.status] ?? 'internal',
    payload?.error ?? `HTTP ${res.status}`,
    Number.isFinite(retry) && retry > 0 ? retry : null,
  );
}

export function createApiClient(options: ApiClientOptions) {
  const { baseUrl, tokens } = options;
  const doFetch = options.fetchImpl ?? ((...args: Parameters<typeof fetch>) => fetch(...args));
  const defaultTimeout = options.timeoutMs ?? 10_000;
  let refreshInFlight: Promise<string> | null = null;
  let onAuthLost: (() => void) | null = null;

  async function rawFetch(
    method: string,
    path: string,
    opts: RequestOptions,
    accessToken: string | null,
  ): Promise<Response> {
    const controller = new AbortController();
    let timedOut = false;
    const timer = setTimeout(() => {
      timedOut = true;
      controller.abort();
    }, opts.timeoutMs ?? defaultTimeout);
    const onAbort = () => controller.abort();
    if (opts.signal) {
      if (opts.signal.aborted) controller.abort();
      else opts.signal.addEventListener('abort', onAbort, { once: true });
    }
    const headers: Record<string, string> = { Accept: 'application/json' };
    if (accessToken) headers.Authorization = `Bearer ${accessToken}`;
    let body: BodyInit | undefined;
    if (opts.form) {
      body = opts.form;
    } else if (opts.body !== undefined) {
      headers['Content-Type'] = 'application/json';
      body = JSON.stringify(opts.body);
    }
    try {
      return await doFetch(`${baseUrl}${path}${buildQuery(opts.query)}`, {
        method,
        headers,
        body,
        signal: controller.signal,
      });
    } catch (err) {
      if (timedOut) throw new ApiError(0, 'timeout', 'timeout');
      if ((err as { name?: string }).name === 'AbortError') throw err;
      throw new ApiError(0, 'network', 'network error');
    } finally {
      clearTimeout(timer);
      opts.signal?.removeEventListener('abort', onAbort);
    }
  }

  async function loseSession(): Promise<never> {
    await tokens.clear();
    onAuthLost?.();
    throw new ApiError(401, 'unauthorized', 'session expired');
  }

  /**
   * Single-flight token refresh. `staleAccess` is the access token that just got a 401: when the
   * stored token already differs, another request refreshed in the meantime and we just reuse it.
   */
  function refresh(staleAccess: string | null): Promise<string> {
    if (refreshInFlight) return refreshInFlight;
    const run = (async () => {
      const current = await tokens.get();
      if (!current) throw new ApiError(401, 'unauthorized', 'no session');
      if (staleAccess && current.accessToken !== staleAccess) return current.accessToken;
      const res = await rawFetch(
        'POST',
        '/auth/refresh',
        { body: { refreshToken: current.refreshToken }, auth: false },
        null,
      );
      if (res.status >= 500) throw await toApiError(res);
      if (!res.ok) return loseSession();
      const data = (await res.json()) as AuthResponse;
      const next: AuthTokens = { accessToken: data.accessToken, refreshToken: data.refreshToken };
      await tokens.set(next);
      return next.accessToken;
    })();
    refreshInFlight = run.finally(() => {
      refreshInFlight = null;
    });
    return refreshInFlight;
  }

  async function send(method: string, path: string, opts: RequestOptions): Promise<Response> {
    const useAuth = opts.auth !== false;
    const session = useAuth ? await tokens.get() : null;
    let res = await rawFetch(method, path, opts, session?.accessToken ?? null);
    if (res.status === 401 && useAuth && session) {
      const fresh = await refresh(session.accessToken);
      // A 401 on a brand-new token is an endpoint-level refusal (e.g. wrong current password):
      // surface it instead of logging out. A truly dead session fails at the next /auth/refresh.
      res = await rawFetch(method, path, opts, fresh);
    }
    if (!res.ok) throw await toApiError(res);
    return res;
  }

  async function json<T>(method: string, path: string, opts: RequestOptions = {}): Promise<T> {
    const res = await send(method, path, opts);
    const text = await res.text();
    return (text ? JSON.parse(text) : undefined) as T;
  }

  return {
    get: <T>(path: string, opts?: RequestOptions) => json<T>('GET', path, opts),
    post: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
      json<T>('POST', path, { ...opts, body }),
    put: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
      json<T>('PUT', path, { ...opts, body }),
    delete: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
      json<T>('DELETE', path, { ...opts, body }),
    upload: <T>(path: string, form: FormData, method: 'POST' | 'PUT' = 'POST') =>
      json<T>(method, path, { form, timeoutMs: 45_000 }),
    /** Authenticated binary download (used by web image loading). */
    async blob(path: string, opts?: RequestOptions): Promise<Blob> {
      const res = await send('GET', path, { ...opts, timeoutMs: opts?.timeoutMs ?? 30_000 });
      return res.blob();
    },
    /** Forces a refresh now (e.g. after a native image failed with an expired token). */
    refreshSession: () => refresh(null).then(() => undefined),
    setAuthLostHandler(handler: (() => void) | null) {
      onAuthLost = handler;
    },
    resolveUrl: (path: string) => (/^https?:/i.test(path) ? path : `${baseUrl}${path}`),
  };
}

export type ApiClient = ReturnType<typeof createApiClient>;

export const api: ApiClient = createApiClient({ baseUrl: API_BASE_URL, tokens: tokenStore });
