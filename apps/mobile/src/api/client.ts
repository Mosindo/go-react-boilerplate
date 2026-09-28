import type { AuthResponse, ErrorCode } from "./types";

export type AuthTokens = { accessToken: string; refreshToken: string };

export type TokenStore = {
  get: () => Promise<AuthTokens | null>;
  save: (tokens: AuthTokens) => Promise<void>;
  clear: () => Promise<void>;
};

const KNOWN_CODES: readonly ErrorCode[] = [
  "invalid_request",
  "unauthorized",
  "forbidden",
  "not_found",
  "conflict",
  "rate_limited",
  "profile_incomplete",
  "underage",
  "blocked",
  "payload_too_large",
  "unsupported_media",
  "internal"
];

function codeFromStatus(status: number): ErrorCode {
  switch (status) {
    case 400:
    case 422:
      return "invalid_request";
    case 401:
      return "unauthorized";
    case 403:
      return "forbidden";
    case 404:
      return "not_found";
    case 409:
      return "conflict";
    case 413:
      return "payload_too_large";
    case 415:
      return "unsupported_media";
    case 429:
      return "rate_limited";
    default:
      return status >= 500 ? "internal" : "unknown";
  }
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: ErrorCode;
  readonly retryAfterSeconds: number | null;

  constructor(status: number, code: ErrorCode, message: string, retryAfterSeconds: number | null = null) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.retryAfterSeconds = retryAfterSeconds;
  }
}

/** Human readable message for any thrown value. */
export function errorMessage(error: unknown, fallback = "Something went wrong. Please try again."): string {
  if (error instanceof ApiError) {
    if (error.code === "rate_limited") {
      return "You're going a little fast. Please wait a moment and try again.";
    }
    return error.message || fallback;
  }
  return fallback;
}

function parseErrorBody(status: number, text: string, retryAfter: string | null): ApiError {
  let message = "";
  let code: ErrorCode = codeFromStatus(status);
  try {
    const payload: unknown = JSON.parse(text);
    if (typeof payload === "object" && payload !== null) {
      const record = payload as Record<string, unknown>;
      if (typeof record.error === "string") {
        message = record.error;
      }
      if (typeof record.code === "string" && (KNOWN_CODES as readonly string[]).includes(record.code)) {
        code = record.code as ErrorCode;
      }
    }
  } catch {
    // Non JSON error bodies fall back to a generic message.
  }
  const seconds = retryAfter ? Number.parseInt(retryAfter, 10) : Number.NaN;
  return new ApiError(status, code, message || `Request failed (${status})`, Number.isFinite(seconds) ? seconds : null);
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  query?: Record<string, string | number | undefined | null>;
  body?: unknown;
  form?: FormData;
  /** Set to false for public endpoints (no bearer token, no refresh on 401). */
  auth?: boolean;
  signal?: AbortSignal;
};

export type ClientDeps = {
  baseUrl: string;
  fetchImpl: typeof fetch;
  tokens: TokenStore;
  onSessionExpired?: () => void;
  timeoutMs?: number;
  uploadTimeoutMs?: number;
  now?: () => number;
};

function decodeJwtExpiryMs(token: string): number | null {
  const part = token.split(".")[1];
  if (!part) {
    return null;
  }
  try {
    const normalized = part.replace(/-/g, "+").replace(/_/g, "/");
    const padded = normalized + "=".repeat((4 - (normalized.length % 4)) % 4);
    const payload: unknown = JSON.parse(atob(padded));
    if (typeof payload === "object" && payload !== null) {
      const exp = (payload as Record<string, unknown>).exp;
      if (typeof exp === "number") {
        return exp * 1000;
      }
    }
  } catch {
    return null;
  }
  return null;
}

const EXPIRY_SKEW_MS = 30_000;

export type ApiClient = {
  baseUrl: string;
  request: <T = void>(path: string, options?: RequestOptions) => Promise<T>;
  /** Returns an access token that is valid for at least a short while, refreshing if needed. */
  getValidAccessToken: () => Promise<string | null>;
};

export function createApiClient(deps: ClientDeps): ApiClient {
  const { baseUrl, fetchImpl, tokens, onSessionExpired } = deps;
  const timeoutMs = deps.timeoutMs ?? 15_000;
  const uploadTimeoutMs = deps.uploadTimeoutMs ?? 60_000;
  const now = deps.now ?? Date.now;
  let refreshInFlight: Promise<string | null> | null = null;

  async function send(path: string, options: RequestOptions, accessToken: string | null): Promise<Response> {
    if (!baseUrl) {
      throw new ApiError(0, "network", "The app is not configured with an API address (EXPO_PUBLIC_API_URL).");
    }
    const query = options.query
      ? Object.entries(options.query)
          .filter((entry): entry is [string, string | number] => entry[1] !== undefined && entry[1] !== null)
          .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
          .join("&")
      : "";
    const url = `${baseUrl}${path}${query ? `?${query}` : ""}`;

    const headers: Record<string, string> = { Accept: "application/json" };
    if (accessToken) {
      headers.Authorization = `Bearer ${accessToken}`;
    }
    let body: string | FormData | undefined;
    if (options.form) {
      body = options.form;
    } else if (options.body !== undefined) {
      headers["Content-Type"] = "application/json";
      body = JSON.stringify(options.body);
    }

    const controller = new AbortController();
    let timedOut = false;
    const timer = setTimeout(
      () => {
        timedOut = true;
        controller.abort();
      },
      options.form ? uploadTimeoutMs : timeoutMs
    );
    const onCallerAbort = () => controller.abort();
    options.signal?.addEventListener("abort", onCallerAbort, { once: true });

    try {
      return await fetchImpl(url, { method: options.method ?? "GET", headers, body, signal: controller.signal });
    } catch (error) {
      if (timedOut) {
        throw new ApiError(0, "timeout", "The request timed out. Please try again.");
      }
      if (options.signal?.aborted) {
        throw error;
      }
      throw new ApiError(0, "network", "Can't reach the server. Check your connection and try again.");
    } finally {
      clearTimeout(timer);
      options.signal?.removeEventListener("abort", onCallerAbort);
    }
  }

  async function parse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      const text = await response.text().catch(() => "");
      throw parseErrorBody(response.status, text, response.headers.get("Retry-After"));
    }
    if (response.status === 204) {
      return undefined as T;
    }
    const text = await response.text();
    return (text ? JSON.parse(text) : undefined) as T;
  }

  /** Single-flight refresh: concurrent callers share one rotation of the refresh token. */
  function refreshAccessToken(staleAccessToken: string | null): Promise<string | null> {
    if (refreshInFlight) {
      return refreshInFlight;
    }
    const run = async (): Promise<string | null> => {
      const current = await tokens.get();
      if (!current) {
        return null;
      }
      if (staleAccessToken && current.accessToken !== staleAccessToken) {
        // Another caller already rotated the tokens after our request was sent.
        return current.accessToken;
      }
      try {
        const response = await send(
          "/auth/refresh",
          { method: "POST", body: { refreshToken: current.refreshToken }, auth: false },
          null
        );
        const session = await parse<AuthResponse>(response);
        await tokens.save({ accessToken: session.accessToken, refreshToken: session.refreshToken });
        return session.accessToken;
      } catch (error) {
        if (error instanceof ApiError && error.status >= 400 && error.status < 500 && error.status !== 429) {
          await tokens.clear();
          onSessionExpired?.();
          return null;
        }
        throw error;
      }
    };
    refreshInFlight = run().finally(() => {
      refreshInFlight = null;
    });
    return refreshInFlight;
  }

  async function request<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
    if (options.auth === false) {
      return parse<T>(await send(path, options, null));
    }
    const stored = await tokens.get();
    const usedToken = stored?.accessToken ?? null;
    const response = await send(path, options, usedToken);
    if (response.status !== 401) {
      return parse<T>(response);
    }
    const fresh = await refreshAccessToken(usedToken);
    if (!fresh) {
      throw new ApiError(401, "unauthorized", "Your session has expired. Please sign in again.");
    }
    return parse<T>(await send(path, options, fresh));
  }

  async function getValidAccessToken(): Promise<string | null> {
    const stored = await tokens.get();
    if (!stored) {
      return null;
    }
    const expiry = decodeJwtExpiryMs(stored.accessToken);
    if (expiry !== null && expiry - now() > EXPIRY_SKEW_MS) {
      return stored.accessToken;
    }
    return refreshAccessToken(stored.accessToken);
  }

  return { baseUrl, request, getValidAccessToken };
}
