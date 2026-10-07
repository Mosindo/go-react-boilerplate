import { Platform } from "react-native";
import { strings } from "../i18n/strings";
import { showGlobalError } from "../shared/feedback/store";
import { clearTokens, getTokens, saveTokens, type AuthTokens } from "../store/tokenStore";
import type { ApiErrorCode, AuthSession } from "./types";

/**
 * The only module allowed to call fetch. UI code goes through the typed functions in
 * src/api/*.ts, which go through apiRequest().
 *
 * - Bearer token is injected automatically (opt out with `auth: false` for public routes).
 * - A 401 triggers ONE single-flight refresh (POST /auth/refresh) and ONE retry of the request.
 *   If the refresh itself is rejected the session is cleared and the handler registered with
 *   setSessionExpiredHandler() fires (AuthProvider signs the user out).
 * - `body` may be a plain object (sent as JSON), a JSON string, or FormData (multipart).
 */

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function developmentFallbackBaseUrl(): string {
  return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }
  return __DEV__ ? developmentFallbackBaseUrl() : "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export class ApiError extends Error {
  public readonly status: number;
  public readonly code: ApiErrorCode | string;
  public readonly data: unknown;
  public readonly retryAfterSeconds: number | null;

  constructor(
    status: number,
    code: ApiErrorCode | string,
    message: string,
    data: unknown = null,
    retryAfterSeconds: number | null = null
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.data = data;
    this.retryAfterSeconds = retryAfterSeconds;
  }
}

export function isApiError(error: unknown, code?: string): error is ApiError {
  return error instanceof ApiError && (code === undefined || error.code === code);
}

/** Best human readable message for any thrown value. */
export function getErrorMessage(error: unknown, fallback: string = strings.errors.generic): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}

export type RequestOptions = Omit<RequestInit, "body" | "headers"> & {
  body?: BodyInit | object | null;
  headers?: Record<string, string>;
  /** Attach the bearer token and refresh on 401. Default true. */
  auth?: boolean;
  timeoutMs?: number;
};

const DEFAULT_TIMEOUT_MS = 15_000;
export const UPLOAD_TIMEOUT_MS = 60_000;

let sessionExpiredHandler: (() => void) | null = null;

export function setSessionExpiredHandler(handler: (() => void) | null): void {
  sessionExpiredHandler = handler;
}

/** Resolves a relative media URL from the API ("/media/...") against the API base URL. */
export function resolveMediaUrl(url: string): string {
  return /^https?:\/\//i.test(url) ? url : `${API_BASE_URL}${url}`;
}

function isFormData(body: unknown): body is FormData {
  return typeof FormData !== "undefined" && body instanceof FormData;
}

function isRawBody(body: unknown): boolean {
  return (
    typeof body === "string" ||
    isFormData(body) ||
    (typeof Blob !== "undefined" && body instanceof Blob) ||
    body instanceof ArrayBuffer
  );
}

function buildRequestInit(options: RequestOptions, token: string | null, signal: AbortSignal): RequestInit {
  const { body, headers: extraHeaders, method, credentials, cache, redirect, referrer, mode } = options;
  const rest: RequestInit = { method, credentials, cache, redirect, referrer, mode };
  const headers: Record<string, string> = { Accept: "application/json", ...extraHeaders };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  let payload: BodyInit | undefined;
  if (body !== undefined && body !== null) {
    if (isRawBody(body)) {
      payload = body as BodyInit;
      // FormData: let fetch set the multipart boundary itself.
      if (typeof body === "string" && !headers["Content-Type"]) {
        headers["Content-Type"] = "application/json";
      }
    } else {
      payload = JSON.stringify(body);
      headers["Content-Type"] = "application/json";
    }
  }
  return { ...rest, headers, body: payload, signal };
}

async function send(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  if (!API_BASE_URL) {
    showGlobalError(strings.errors.noApiUrl);
    throw new ApiError(0, "network_error", strings.errors.noApiUrl);
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), options.timeoutMs ?? DEFAULT_TIMEOUT_MS);
  const external = options.signal;
  const onExternalAbort = () => controller.abort();
  if (external) {
    if (external.aborted) {
      controller.abort();
    } else {
      external.addEventListener("abort", onExternalAbort, { once: true });
    }
  }

  try {
    return await fetch(`${API_BASE_URL}${path}`, buildRequestInit(options, token, controller.signal));
  } catch (error) {
    if (external?.aborted) {
      throw error;
    }
    if ((error as { name?: string }).name === "AbortError") {
      showGlobalError(strings.errors.timeout);
      throw new ApiError(0, "timeout", strings.errors.timeout);
    }
    showGlobalError(strings.errors.network);
    throw new ApiError(0, "network_error", strings.errors.network);
  } finally {
    clearTimeout(timeoutId);
    external?.removeEventListener("abort", onExternalAbort);
  }
}

function defaultCode(status: number): string {
  switch (status) {
    case 400:
      return "validation_error";
    case 401:
      return "unauthorized";
    case 403:
      return "forbidden";
    case 404:
      return "not_found";
    case 409:
      return "conflict";
    case 429:
      return "rate_limited";
    default:
      return status >= 500 ? "internal" : "unknown";
  }
}

async function toApiError(response: Response): Promise<ApiError> {
  let data: unknown = null;
  try {
    const text = await response.text();
    data = text ? (JSON.parse(text) as unknown) : null;
  } catch {
    data = null;
  }
  const payload = (typeof data === "object" && data !== null ? data : {}) as {
    error?: unknown;
    code?: unknown;
  };
  const code = typeof payload.code === "string" ? payload.code : defaultCode(response.status);
  let message = typeof payload.error === "string" && payload.error ? payload.error : "";
  const retryHeader = Number(response.headers?.get?.("Retry-After"));
  const retryAfter = Number.isFinite(retryHeader) && retryHeader > 0 ? retryHeader : null;

  if (response.status === 429) {
    message = message || strings.errors.rateLimited;
  } else if (response.status >= 500) {
    message = strings.errors.server;
  } else if (!message) {
    message = strings.errors.generic;
  }
  if (response.status >= 500 || response.status === 429) {
    showGlobalError(message);
  }
  return new ApiError(response.status, code, message, data, retryAfter);
}

let refreshInFlight: Promise<string | null> | null = null;

async function performRefresh(): Promise<string | null> {
  const tokens = await getTokens();
  if (!tokens?.refreshToken) {
    return null;
  }

  const response = await send(
    "/auth/refresh",
    { method: "POST", body: { refreshToken: tokens.refreshToken } },
    null
  );
  if (response.ok) {
    const session = (await response.json()) as AuthSession;
    await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    return session.accessToken;
  }
  if (response.status >= 500 || response.status === 429) {
    // Transient: keep the session, let the caller see an error.
    throw await toApiError(response);
  }
  // The server rejected our refresh token: the session is over.
  await clearTokens();
  sessionExpiredHandler?.();
  return null;
}

/**
 * Returns a fresh access token. If another request already refreshed (stored token differs from
 * the one that failed) it is reused; concurrent callers share a single refresh call.
 */
async function refreshAccessToken(failedToken: string): Promise<string | null> {
  const current: AuthTokens | null = await getTokens();
  if (current && current.accessToken !== failedToken) {
    return current.accessToken;
  }
  if (!refreshInFlight) {
    refreshInFlight = performRefresh().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

export async function apiRequest<T = unknown>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  const tokens = useAuth ? await getTokens() : null;
  let response = await send(path, options, tokens?.accessToken ?? null);

  if (response.status === 401 && useAuth && tokens?.accessToken) {
    const fresh = await refreshAccessToken(tokens.accessToken);
    if (fresh) {
      response = await send(path, options, fresh);
    }
  }

  if (!response.ok) {
    throw await toApiError(response);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

/** Test helper: forget any in-flight refresh between tests. */
export function resetClientStateForTests(): void {
  refreshInFlight = null;
  sessionExpiredHandler = null;
}
