import type { AuthResponse } from "./types";

export type TokenPair = { accessToken: string; refreshToken: string };

export type TokenStorage = {
  get(): TokenPair | null;
  set(tokens: TokenPair): Promise<void>;
  clear(): Promise<void>;
};

export type UploadFile = { uri: string; name: string; type: string };

export type QueryParams = Record<string, string | number | boolean | null | undefined>;

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  query?: QueryParams;
  /** Plain objects are sent as JSON, FormData is sent as is. */
  body?: Record<string, unknown> | FormData;
  signal?: AbortSignal;
  timeoutMs?: number;
  /** Set to false for endpoints that must not carry (or refresh) credentials. */
  auth?: boolean;
};

export class ApiError extends Error {
  public readonly status: number;
  public readonly code: string;
  public readonly data: unknown;

  constructor(status: number, code: string, message: string, data: unknown = null) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.data = data;
  }
}

export type HttpClientConfig = {
  baseUrl: string;
  storage: TokenStorage;
  fetchImpl?: typeof fetch;
  timeoutMs?: number;
  onSessionExpired?: () => void;
  now?: () => number;
};

const DEFAULT_TIMEOUT_MS = 10_000;
export const UPLOAD_TIMEOUT_MS = 30_000;
const TOKEN_LEEWAY_SECONDS = 30;

export function buildQuery(params: QueryParams | undefined): string {
  if (!params) {
    return "";
  }
  const parts: string[] = [];
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null) {
      parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`);
    }
  }
  return parts.length > 0 ? `?${parts.join("&")}` : "";
}

function fallbackCode(status: number): string {
  if (status >= 500) return "internal";
  if (status === 400) return "invalid_request";
  if (status === 401) return "unauthorized";
  if (status === 403) return "forbidden";
  if (status === 404) return "not_found";
  if (status === 409) return "conflict";
  if (status === 413) return "payload_too_large";
  if (status === 429) return "rate_limited";
  return "unknown";
}

function toApiError(status: number, data: unknown): ApiError {
  const payload = typeof data === "object" && data !== null ? (data as { error?: unknown; code?: unknown }) : {};
  const message = typeof payload.error === "string" ? payload.error : `API request failed (${status})`;
  const code = typeof payload.code === "string" ? payload.code : fallbackCode(status);
  return new ApiError(status, code, message, data);
}

/** Reads the `exp` claim (seconds) of a JWT without verifying it; null when unreadable. */
export function readTokenExpiry(token: string): number | null {
  try {
    const payload = token.split(".")[1];
    if (!payload) {
      return null;
    }
    const base64 = payload.replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
    const parsed = JSON.parse(atob(padded)) as { exp?: unknown };
    return typeof parsed.exp === "number" ? parsed.exp : null;
  } catch {
    return null;
  }
}

/** A 401 means "bad token" except on endpoints that verify a password (the code tells them apart). */
function isTokenRejection(error: unknown): error is ApiError {
  return error instanceof ApiError && error.status === 401 && error.code !== "invalid_credentials";
}

export type HttpClient = ReturnType<typeof createHttpClient>;

export function createHttpClient(config: HttpClientConfig) {
  const fetchImpl = config.fetchImpl ?? ((input, init) => fetch(input, init));
  const now = config.now ?? (() => Date.now());
  let refreshInFlight: Promise<TokenPair> | null = null;

  async function send<T>(path: string, options: RequestOptions, token: string | null): Promise<T> {
    if (!config.baseUrl) {
      throw new ApiError(0, "config", "L'adresse du serveur n'est pas configurée (EXPO_PUBLIC_API_URL).");
    }

    const controller = new AbortController();
    const timeoutMs = options.timeoutMs ?? config.timeoutMs ?? DEFAULT_TIMEOUT_MS;
    let timedOut = false;
    const timeoutId = setTimeout(() => {
      timedOut = true;
      controller.abort();
    }, timeoutMs);
    const onExternalAbort = () => controller.abort();
    if (options.signal) {
      if (options.signal.aborted) {
        controller.abort();
      } else {
        options.signal.addEventListener("abort", onExternalAbort, { once: true });
      }
    }

    const headers: Record<string, string> = { Accept: "application/json" };
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
    let body: BodyInit | undefined;
    if (options.body instanceof FormData) {
      body = options.body;
    } else if (options.body) {
      headers["Content-Type"] = "application/json";
      body = JSON.stringify(options.body);
    }

    try {
      const response = await fetchImpl(`${config.baseUrl}${path}${buildQuery(options.query)}`, {
        method: options.method ?? "GET",
        headers,
        body,
        signal: controller.signal
      });
      const text = await response.text();
      let data: unknown = null;
      if (text) {
        try {
          data = JSON.parse(text);
        } catch {
          data = text;
        }
      }
      if (!response.ok) {
        throw toApiError(response.status, data);
      }
      return data as T;
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }
      if (timedOut) {
        throw new ApiError(0, "timeout", "Le serveur met trop de temps à répondre.");
      }
      if (options.signal?.aborted) {
        throw error;
      }
      throw new ApiError(0, "network_error", "Connexion au serveur impossible.");
    } finally {
      clearTimeout(timeoutId);
      options.signal?.removeEventListener("abort", onExternalAbort);
    }
  }

  async function runRefresh(): Promise<TokenPair> {
    const stored = config.storage.get();
    if (!stored) {
      throw new ApiError(401, "unauthorized", "Session expirée.");
    }
    try {
      const response = await send<AuthResponse>(
        "/auth/refresh",
        { method: "POST", body: { refreshToken: stored.refreshToken }, auth: false },
        null
      );
      const pair: TokenPair = { accessToken: response.accessToken, refreshToken: response.refreshToken };
      await config.storage.set(pair);
      return pair;
    } catch (error) {
      const rejected =
        error instanceof ApiError &&
        error.status >= 400 &&
        error.status < 500 &&
        error.status !== 408 &&
        error.status !== 429;
      if (rejected) {
        await config.storage.clear();
        config.onSessionExpired?.();
      }
      throw error;
    }
  }

  /** Single-flight: concurrent callers share one refresh call (the refresh token rotates). */
  function refreshTokens(): Promise<TokenPair> {
    if (!refreshInFlight) {
      refreshInFlight = runRefresh().finally(() => {
        refreshInFlight = null;
      });
    }
    return refreshInFlight;
  }

  async function request<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
    if (options.auth === false) {
      return send<T>(path, options, null);
    }

    const sentWith = config.storage.get()?.accessToken ?? null;
    try {
      return await send<T>(path, options, sentWith);
    } catch (error) {
      if (!isTokenRejection(error) || !sentWith) {
        throw error;
      }
    }

    const latest = config.storage.get();
    if (!latest) {
      throw new ApiError(401, "unauthorized", "Session expirée.");
    }
    // Another request may already have rotated the tokens while this one was in flight.
    const token = latest.accessToken !== sentWith ? latest.accessToken : (await refreshTokens()).accessToken;
    try {
      return await send<T>(path, options, token);
    } catch (error) {
      if (isTokenRejection(error)) {
        await config.storage.clear();
        config.onSessionExpired?.();
      }
      throw error;
    }
  }

  /** Returns an access token valid for at least a few more seconds, refreshing when needed. */
  async function getFreshAccessToken(force = false): Promise<string | null> {
    const stored = config.storage.get();
    if (!stored) {
      return null;
    }
    const expiry = readTokenExpiry(stored.accessToken);
    const stale = expiry !== null && expiry - now() / 1000 < TOKEN_LEEWAY_SECONDS;
    if (!force && !stale) {
      return stored.accessToken;
    }
    return (await refreshTokens()).accessToken;
  }

  return {
    request,
    getFreshAccessToken,
    get: <T>(path: string, query?: QueryParams, signal?: AbortSignal) => request<T>(path, { query, signal }),
    post: <T = void>(path: string, body?: Record<string, unknown>, options?: Pick<RequestOptions, "auth">) =>
      request<T>(path, { method: "POST", body, ...options }),
    put: <T = void>(path: string, body?: Record<string, unknown>) => request<T>(path, { method: "PUT", body }),
    delete: <T = void>(path: string, body?: Record<string, unknown>) => request<T>(path, { method: "DELETE", body }),
    upload: <T>(path: string, method: "POST" | "PUT", file: UploadFile) => {
      const form = new FormData();
      // React Native's FormData accepts a {uri, name, type} descriptor in place of a Blob.
      form.append("file", file as unknown as Blob);
      return request<T>(path, { method, body: form, timeoutMs: UPLOAD_TIMEOUT_MS });
    }
  };
}
