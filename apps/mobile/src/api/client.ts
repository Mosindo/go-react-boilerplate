import { API_BASE_URL } from "./config";
import type { ApiErrorCode, AuthResponse } from "./types";
import { tokenStore, type TokenStore } from "../store/tokenStore";

export class ApiError extends Error {
  readonly status: number;
  readonly code: ApiErrorCode;
  readonly fields: Record<string, string>;
  /** Seconds, from the Retry-After header on 429. */
  readonly retryAfter: number | null;

  constructor(
    status: number,
    code: ApiErrorCode,
    message: string,
    fields: Record<string, string> = {},
    retryAfter: number | null = null
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.fields = fields;
    this.retryAfter = retryAfter;
  }
}

export function isApiError(value: unknown): value is ApiError {
  return value instanceof ApiError;
}

const CODE_BY_STATUS: Record<number, ApiErrorCode> = {
  400: "validation_failed",
  401: "unauthorized",
  403: "forbidden",
  404: "not_found",
  409: "conflict",
  413: "payload_too_large",
  422: "validation_failed",
  429: "rate_limited"
};

const KNOWN_CODES: ReadonlySet<string> = new Set<ApiErrorCode>([
  "validation_failed",
  "unauthorized",
  "forbidden",
  "not_found",
  "conflict",
  "profile_incomplete",
  "rate_limited",
  "payload_too_large",
  "internal"
]);

/** Maps an HTTP status + decoded body to a typed ApiError (exported for tests). */
export function mapErrorResponse(status: number, body: unknown, retryAfterHeader?: string | null): ApiError {
  const payload = typeof body === "object" && body !== null ? (body as Record<string, unknown>) : {};
  const rawCode = typeof payload.code === "string" ? payload.code : "";
  const code: ApiErrorCode = KNOWN_CODES.has(rawCode)
    ? (rawCode as ApiErrorCode)
    : (CODE_BY_STATUS[status] ?? (status >= 500 ? "internal" : "unknown"));
  const message =
    typeof payload.error === "string" && payload.error
      ? payload.error
      : status >= 500
        ? "The server had a problem. Please try again in a moment."
        : `Request failed (${status})`;
  const fields: Record<string, string> = {};
  if (typeof payload.fields === "object" && payload.fields !== null) {
    for (const [key, value] of Object.entries(payload.fields)) {
      if (typeof value === "string") fields[key] = value;
    }
  }
  const retry = retryAfterHeader ? Number.parseInt(retryAfterHeader, 10) : NaN;
  return new ApiError(status, code, message, fields, Number.isFinite(retry) ? retry : null);
}

/** Human readable message for any thrown value. */
export function getErrorMessage(error: unknown, fallback = "Something went wrong. Please try again."): string {
  if (isApiError(error)) {
    if (error.code === "rate_limited") {
      return error.retryAfter
        ? `Too many attempts. Please wait ${error.retryAfter}s and try again.`
        : "Too many attempts. Please wait a moment and try again.";
    }
    return error.message;
  }
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

type Query = Record<string, string | number | boolean | undefined | null>;

export type RequestOptions = {
  query?: Query;
  body?: unknown;
  formData?: FormData;
  /** Set false for endpoints that must not carry (or refresh) credentials. */
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

export type ApiClientOptions = {
  baseUrl: string;
  tokens: TokenStore;
  fetchImpl?: typeof fetch;
  onAuthLost?: () => void;
  timeoutMs?: number;
};

export class ApiClient {
  readonly baseUrl: string;
  readonly tokens: TokenStore;
  private readonly fetchImpl: typeof fetch;
  private readonly timeoutMs: number;
  private onAuthLost: (() => void) | undefined;
  private refreshInFlight: Promise<boolean> | null = null;

  constructor(options: ApiClientOptions) {
    this.baseUrl = options.baseUrl;
    this.tokens = options.tokens;
    this.fetchImpl = options.fetchImpl ?? ((...args) => fetch(...args));
    this.onAuthLost = options.onAuthLost;
    this.timeoutMs = options.timeoutMs ?? 15_000;
  }

  setAuthLostHandler(handler: (() => void) | undefined) {
    this.onAuthLost = handler;
  }

  get<T>(path: string, options?: RequestOptions) {
    return this.request<T>("GET", path, options);
  }
  post<T>(path: string, body?: unknown, options?: RequestOptions) {
    return this.request<T>("POST", path, { ...options, body });
  }
  put<T>(path: string, body?: unknown, options?: RequestOptions) {
    return this.request<T>("PUT", path, { ...options, body });
  }
  delete<T>(path: string, body?: unknown, options?: RequestOptions) {
    return this.request<T>("DELETE", path, { ...options, body });
  }

  async request<T>(method: string, path: string, options: RequestOptions = {}): Promise<T> {
    const useAuth = options.auth !== false;
    const usedToken = useAuth ? this.tokens.getAccessToken() : null;

    let response = await this.send(method, path, options, usedToken);

    if (response.status === 401 && useAuth) {
      const refreshed = await this.refreshSession(usedToken);
      if (refreshed) {
        response = await this.send(method, path, options, this.tokens.getAccessToken());
      }
    }

    if (!response.ok) {
      let body: unknown = null;
      try {
        body = await response.json();
      } catch {
        body = null;
      }
      throw mapErrorResponse(response.status, body, response.headers.get("retry-after"));
    }

    if (response.status === 204) return undefined as T;
    const text = await response.text();
    return (text ? JSON.parse(text) : undefined) as T;
  }

  /**
   * Refreshes the session. Concurrent callers share one network call (single flight).
   * If the token already changed since `staleToken` was used, no new refresh is needed.
   * Returns false when the session is gone (and notifies via onAuthLost).
   */
  refreshSession(staleToken: string | null = null): Promise<boolean> {
    const current = this.tokens.getAccessToken();
    if (staleToken !== null && current !== null && current !== staleToken && !this.refreshInFlight) {
      return Promise.resolve(true);
    }
    if (!this.refreshInFlight) {
      this.refreshInFlight = this.doRefresh().finally(() => {
        this.refreshInFlight = null;
      });
    }
    return this.refreshInFlight;
  }

  private async doRefresh(): Promise<boolean> {
    const refreshToken = this.tokens.getTokens()?.refreshToken;
    if (!refreshToken) {
      await this.dropSession();
      return false;
    }
    try {
      const data = await this.request<AuthResponse>("POST", "/auth/refresh", {
        auth: false,
        body: { refreshToken }
      });
      await this.tokens.set({ accessToken: data.accessToken, refreshToken: data.refreshToken });
      return true;
    } catch (error) {
      // Network problems must not sign the user out; an explicit rejection does.
      if (isApiError(error) && error.status >= 400 && error.status < 500 && error.status !== 429) {
        await this.dropSession();
        return false;
      }
      throw error;
    }
  }

  private async dropSession() {
    await this.tokens.clear();
    this.onAuthLost?.();
  }

  private async send(method: string, path: string, options: RequestOptions, token: string | null): Promise<Response> {
    const controller = new AbortController();
    let timedOut = false;
    const timeout = setTimeout(() => {
      timedOut = true;
      controller.abort();
    }, options.timeoutMs ?? this.timeoutMs);
    const onAbort = () => controller.abort();
    options.signal?.addEventListener("abort", onAbort, { once: true });

    const headers: Record<string, string> = { Accept: "application/json" };
    if (token) headers.Authorization = `Bearer ${token}`;
    let body: BodyInit | undefined;
    if (options.formData) {
      body = options.formData; // the runtime sets the multipart boundary
    } else if (options.body !== undefined) {
      headers["Content-Type"] = "application/json";
      body = JSON.stringify(options.body);
    }

    try {
      return await this.fetchImpl(this.buildUrl(path, options.query), {
        method,
        headers,
        body,
        signal: controller.signal
      });
    } catch (error) {
      if (timedOut) {
        throw new ApiError(0, "timeout", "The request timed out. Check your connection and try again.");
      }
      if (options.signal?.aborted) throw error;
      throw new ApiError(0, "network", "Can't reach Lumen right now. Check your connection and try again.");
    } finally {
      clearTimeout(timeout);
      options.signal?.removeEventListener("abort", onAbort);
    }
  }

  private buildUrl(path: string, query?: Query): string {
    if (!query) return `${this.baseUrl}${path}`;
    const params = Object.entries(query)
      .filter(([, value]) => value !== undefined && value !== null && value !== "")
      .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
      .join("&");
    return params ? `${this.baseUrl}${path}?${params}` : `${this.baseUrl}${path}`;
  }
}

export const api = new ApiClient({ baseUrl: API_BASE_URL, tokens: tokenStore });
