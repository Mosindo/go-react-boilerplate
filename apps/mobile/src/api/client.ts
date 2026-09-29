import { Platform } from "react-native";
import { parseRetryAfter } from "../lib/errors";
import { strings } from "../lib/strings";
import { showGlobalError } from "../shared/feedback/store";
import { clearTokens, getTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

export type ApiErrorPayload = {
  error?: string;
};

export type ApiRequestOptions = RequestInit & {
  /** Abort the request after this many milliseconds. Default 15 s (60 s for multipart bodies). */
  timeoutMs?: number;
  /**
   * Set to false for public endpoints (login, register, forgot, reset, refresh): no bearer token is sent and a 401
   * is a plain error instead of a session-expiry signal.
   */
  authenticated?: boolean;
};

const DEFAULT_TIMEOUT_MS = 15_000;
const UPLOAD_TIMEOUT_MS = 60_000;

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function developmentFallbackBaseUrl(): string {
  return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
}

function resolveApiBaseUrl(): string {
  const explicitBaseUrl = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicitBaseUrl) {
    return normalizeBaseUrl(explicitBaseUrl);
  }

  if (__DEV__) {
    return developmentFallbackBaseUrl();
  }

  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** Photo URLs from the API are relative and signed; this makes them renderable. */
export function resolvePhotoUrl(relativeUrl: string): string {
  if (/^https?:\/\//i.test(relativeUrl)) {
    return relativeUrl;
  }
  return `${API_BASE_URL}${relativeUrl.startsWith("/") ? "" : "/"}${relativeUrl}`;
}

export class ApiError extends Error {
  public status: number;
  public data: unknown;
  public retryAfterSeconds?: number;

  constructor(status: number, data: unknown, message?: string, retryAfterSeconds?: number) {
    super(message ?? `API request failed (${status})`);
    this.name = "ApiError";
    this.status = status;
    this.data = data;
    this.retryAfterSeconds = retryAfterSeconds;
  }
}

/* ------------------------------------------------------------------ session hooks (used by useAuth) */

type AuthHandlers = {
  onTokensRefreshed?: (tokens: AuthTokens) => void;
  onSessionExpired?: () => void;
};

let authHandlers: AuthHandlers = {};

/** The auth provider registers callbacks here; returns an unregister function. */
export function registerAuthHandlers(handlers: AuthHandlers): () => void {
  authHandlers = handlers;
  return () => {
    if (authHandlers === handlers) {
      authHandlers = {};
    }
  };
}

/* ------------------------------------------------------------------ helpers */

function formatNetworkHint(): string {
  if (process.env.EXPO_PUBLIC_API_URL) {
    return strings.errors.network;
  }

  if (!__DEV__) {
    return "API base URL is not configured for this build. Set EXPO_PUBLIC_API_URL and try again.";
  }

  if (Platform.OS === "android") {
    return "Network error. Android emulators usually need 10.0.2.2, and physical devices need EXPO_PUBLIC_API_URL set to your LAN IP.";
  }

  if (Platform.OS === "ios") {
    return "Network error. iOS simulators can use localhost, but physical devices need EXPO_PUBLIC_API_URL set to your LAN IP.";
  }

  return strings.errors.network;
}

function requireApiBaseUrl(): string {
  if (API_BASE_URL) {
    return API_BASE_URL;
  }

  const message = "API base URL is not configured. Set EXPO_PUBLIC_API_URL for this build.";
  showGlobalError(message);
  throw new Error(message);
}

function timeoutError(): Error {
  const error = new Error(strings.errors.timeout);
  error.name = "TimeoutError";
  return error;
}

function isMultipart(body: BodyInit | null | undefined): boolean {
  return typeof FormData !== "undefined" && body instanceof FormData;
}

type Attempt = { response: Response };

async function fetchWithTimeout(
  url: string,
  init: RequestInit,
  timeoutMs: number,
  callerSignal: AbortSignal | null | undefined
): Promise<Attempt> {
  const controller = new AbortController();
  let timedOut = false;
  const timeoutId = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);
  const onCallerAbort = () => controller.abort();
  if (callerSignal) {
    if (callerSignal.aborted) {
      controller.abort();
    } else {
      callerSignal.addEventListener("abort", onCallerAbort, { once: true });
    }
  }
  try {
    const response = await fetch(url, { ...init, signal: controller.signal });
    return { response };
  } catch (error) {
    if (timedOut) {
      throw timeoutError();
    }
    throw error;
  } finally {
    clearTimeout(timeoutId);
    callerSignal?.removeEventListener("abort", onCallerAbort);
  }
}

async function readErrorBody(response: Response): Promise<unknown> {
  const contentType = response.headers.get("content-type") ?? "";
  try {
    return contentType.includes("application/json") ? await response.json() : await response.text();
  } catch {
    return null;
  }
}

function buildApiError(response: Response, data: unknown): ApiError {
  const payload = typeof data === "object" && data !== null ? (data as ApiErrorPayload) : null;
  const retryAfter = parseRetryAfter(response.headers.get("retry-after"));
  let message = payload?.error ?? `API request failed (${response.status})`;
  if (response.status === 429) {
    message =
      retryAfter && retryAfter > 0
        ? `Too many attempts. Please wait ${retryAfter} second${retryAfter === 1 ? "" : "s"} and try again.`
        : strings.errors.rateLimited;
  }
  return new ApiError(response.status, data, message, retryAfter ?? undefined);
}

/* ------------------------------------------------------------------ single-flight refresh */

type RefreshOutcome =
  | { kind: "ok"; accessToken: string }
  | { kind: "rejected" }
  | { kind: "unavailable"; error: unknown };

let refreshInFlight: Promise<RefreshOutcome> | null = null;

type RefreshPayload = { accessToken?: string; token?: string; refreshToken?: string };

async function performRefresh(): Promise<RefreshOutcome> {
  const stored = await getTokens();
  if (!stored?.refreshToken) {
    return { kind: "rejected" };
  }
  try {
    const { response } = await fetchWithTimeout(
      `${requireApiBaseUrl()}/auth/refresh`,
      {
        method: "POST",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify({ refreshToken: stored.refreshToken })
      },
      DEFAULT_TIMEOUT_MS,
      null
    );
    if (response.ok) {
      const payload = (await response.json()) as RefreshPayload;
      const accessToken = payload.accessToken ?? payload.token;
      if (!accessToken || !payload.refreshToken) {
        return { kind: "rejected" };
      }
      const tokens: AuthTokens = { accessToken, refreshToken: payload.refreshToken };
      await saveTokens(tokens);
      authHandlers.onTokensRefreshed?.(tokens);
      return { kind: "ok", accessToken };
    }
    if (response.status >= 500 || response.status === 429) {
      return { kind: "unavailable", error: buildApiError(response, await readErrorBody(response)) };
    }
    return { kind: "rejected" };
  } catch (error) {
    return { kind: "unavailable", error };
  }
}

async function refreshAccessToken(failedAccessToken: string | null): Promise<RefreshOutcome> {
  // Another request may already have rotated the tokens while this one was in flight.
  const current = await getTokens();
  if (current && failedAccessToken && current.accessToken !== failedAccessToken) {
    return { kind: "ok", accessToken: current.accessToken };
  }
  if (!refreshInFlight) {
    refreshInFlight = performRefresh().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

/* ------------------------------------------------------------------ public API */

function buildHeaders(base: HeadersInit | undefined, body: BodyInit | null | undefined, token: string | null): Headers {
  const headers = new Headers(base);
  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  if (body !== undefined && body !== null && !isMultipart(body) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  return headers;
}

export async function apiRequest<T = unknown>(path: string, options?: ApiRequestOptions): Promise<T> {
  const { authenticated = true, timeoutMs, headers: baseHeaders, signal, ...init } = options ?? {};
  const effectiveTimeout = timeoutMs ?? (isMultipart(init.body) ? UPLOAD_TIMEOUT_MS : DEFAULT_TIMEOUT_MS);
  const url = `${requireApiBaseUrl()}${path}`;

  const send = (token: string | null) =>
    fetchWithTimeout(
      url,
      { ...init, headers: buildHeaders(baseHeaders, init.body, token) },
      effectiveTimeout,
      signal
    );

  try {
    let token = authenticated ? ((await getTokens())?.accessToken ?? null) : null;
    let { response } = await send(token);

    if (response.status === 401 && authenticated && token) {
      const outcome = await refreshAccessToken(token);
      if (outcome.kind === "ok") {
        token = outcome.accessToken;
        ({ response } = await send(token));
      } else if (outcome.kind === "rejected") {
        await clearTokens();
        authHandlers.onSessionExpired?.();
        throw new ApiError(401, null, strings.errors.unauthorized);
      } else {
        throw outcome.error;
      }
    }

    if (!response.ok) {
      const error = buildApiError(response, await readErrorBody(response));
      if (response.status >= 500) {
        showGlobalError(strings.errors.server);
      }
      // A 401 that survives a successful token refresh is a business answer (e.g. wrong current password),
      // not a dead session: the user stays signed in. Dead sessions are handled by the refresh outcome above.
      throw error;
    }

    const text = await response.text();
    if (!text) {
      return undefined as T;
    }
    return JSON.parse(text) as T;
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    const name = (error as { name?: string }).name;
    if (name === "TimeoutError") {
      showGlobalError(strings.errors.timeout);
    } else if (error instanceof TypeError) {
      showGlobalError(formatNetworkHint());
    }
    throw error;
  }
}
