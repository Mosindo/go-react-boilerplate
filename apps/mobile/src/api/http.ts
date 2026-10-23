import { ApiError, mapErrorResponse, NETWORK_ERROR_CODE, TIMEOUT_ERROR_CODE } from "./errors";
import type { AuthResponse } from "./types";

export type TokenPair = { accessToken: string; refreshToken: string };

export interface TokenStorage {
  get(): Promise<TokenPair | null>;
  set(tokens: TokenPair): Promise<void>;
  clear(): Promise<void>;
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  query?: Record<string, string | number | undefined | null>;
  body?: unknown;
  formData?: FormData;
  /** false for public endpoints (login, register...). */
  auth?: boolean;
  timeoutMs?: number;
  signal?: AbortSignal;
};

export type HttpClientConfig = {
  baseUrl: string;
  storage: TokenStorage;
  fetchImpl?: typeof fetch;
  /** Called after the refresh token was rejected (or missing): the app must sign out. */
  onSessionExpired?: () => void;
  /** Called after tokens were rotated successfully. */
  onTokensRefreshed?: (tokens: TokenPair) => void;
  defaultTimeoutMs?: number;
};

const DEFAULT_TIMEOUT_MS = 15_000;

export function buildQuery(query: RequestOptions["query"]): string {
  if (!query) return "";
  const parts: string[] = [];
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === "") continue;
    parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`);
  }
  return parts.length ? `?${parts.join("&")}` : "";
}

/** Reads the `exp` claim (seconds) of a JWT without verifying it. */
export function jwtExpiry(token: string): number | null {
  const segment = token.split(".")[1];
  if (!segment) return null;
  try {
    const normalized = segment.replace(/-/g, "+").replace(/_/g, "/");
    const padded = normalized + "=".repeat((4 - (normalized.length % 4)) % 4);
    const payload = JSON.parse(atob(padded)) as { exp?: unknown };
    return typeof payload.exp === "number" ? payload.exp : null;
  } catch {
    return null;
  }
}

export function createHttpClient(config: HttpClientConfig) {
  const doFetch: typeof fetch = config.fetchImpl ?? ((...args) => fetch(...args));
  let refreshInFlight: Promise<TokenPair> | null = null;
  let memoryAccessToken: string | null = null;

  async function send(path: string, options: RequestOptions, accessToken: string | null) {
    const controller = new AbortController();
    const timeout = setTimeout(
      () => controller.abort(),
      options.timeoutMs ?? config.defaultTimeoutMs ?? DEFAULT_TIMEOUT_MS
    );
    if (options.signal) {
      if (options.signal.aborted) controller.abort();
      else options.signal.addEventListener("abort", () => controller.abort(), { once: true });
    }
    const headers: Record<string, string> = { Accept: "application/json" };
    let body: BodyInit | undefined;
    if (options.formData) {
      body = options.formData;
    } else if (options.body !== undefined) {
      headers["Content-Type"] = "application/json";
      body = JSON.stringify(options.body);
    }
    if (accessToken) headers.Authorization = `Bearer ${accessToken}`;
    try {
      return await doFetch(`${config.baseUrl}${path}${buildQuery(options.query)}`, {
        method: options.method ?? "GET",
        headers,
        body,
        signal: controller.signal
      });
    } catch (error) {
      if (controller.signal.aborted && !options.signal?.aborted) {
        throw new ApiError(0, TIMEOUT_ERROR_CODE, "request timed out");
      }
      if (options.signal?.aborted) throw error;
      throw new ApiError(0, NETWORK_ERROR_CODE, "network error");
    } finally {
      clearTimeout(timeout);
    }
  }

  async function parse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      let body: unknown = null;
      try {
        body = await response.json();
      } catch {
        body = null;
      }
      throw mapErrorResponse(response.status, body, response.headers.get("Retry-After"));
    }
    if (response.status === 204) return undefined as T;
    const text = await response.text();
    return (text ? JSON.parse(text) : undefined) as T;
  }

  /** Rotates the token pair. Concurrent callers share one network call (single-flight). */
  function refreshTokens(): Promise<TokenPair> {
    if (refreshInFlight) return refreshInFlight;
    const run = async (): Promise<TokenPair> => {
      const current = await config.storage.get();
      if (!current?.refreshToken) {
        await expire();
        throw new ApiError(401, "unauthorized", "no refresh token");
      }
      try {
        const response = await send(
          "/auth/refresh",
          { method: "POST", body: { refreshToken: current.refreshToken }, auth: false },
          null
        );
        const data = await parse<AuthResponse>(response);
        const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
        memoryAccessToken = next.accessToken;
        await config.storage.set(next);
        config.onTokensRefreshed?.(next);
        return next;
      } catch (error) {
        // Only an explicit rejection ends the session; a network blip must not log the user out.
        if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
          await expire();
        }
        throw error;
      }
    };
    refreshInFlight = run().finally(() => {
      refreshInFlight = null;
    });
    return refreshInFlight;
  }

  async function expire() {
    memoryAccessToken = null;
    await config.storage.clear();
    config.onSessionExpired?.();
  }

  async function request<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
    const needsAuth = options.auth !== false;
    if (!needsAuth) return parse<T>(await send(path, options, null));

    const stored = await config.storage.get();
    memoryAccessToken = stored?.accessToken ?? null;
    const usedToken = stored?.accessToken ?? null;
    let response = await send(path, options, usedToken);
    if (response.status === 401) {
      const latest = await config.storage.get();
      // Another request may already have rotated the tokens while ours was in flight.
      const nextToken =
        latest?.accessToken && latest.accessToken !== usedToken
          ? latest.accessToken
          : (await refreshTokens()).accessToken;
      response = await send(path, options, nextToken);
    }
    return parse<T>(response);
  }

  /** Refreshes ahead of time when the access token expires within `skewSeconds`. */
  async function getFreshAccessToken(skewSeconds = 60): Promise<string | null> {
    const stored = await config.storage.get();
    if (!stored) return null;
    const exp = jwtExpiry(stored.accessToken);
    if (exp !== null && exp * 1000 - Date.now() < skewSeconds * 1000) {
      return (await refreshTokens()).accessToken;
    }
    return stored.accessToken;
  }

  return {
    request,
    refreshTokens,
    getFreshAccessToken,
    getCachedAccessToken: () => memoryAccessToken,
    setCachedAccessToken: (token: string | null) => {
      memoryAccessToken = token;
    }
  };
}

export type HttpClient = ReturnType<typeof createHttpClient>;
