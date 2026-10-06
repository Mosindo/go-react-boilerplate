import { Platform } from "react-native";
import { isExpiringSoon } from "../lib/jwt";
import { clearTokens, getTokens, saveTokens, type AuthTokens } from "../store/tokenStore";
import { endpoints } from "./endpoints";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }
  if (__DEV__) {
    // Emulator loopbacks only. A physical phone needs EXPO_PUBLIC_API_URL=http://<LAN_IP>:<PORT>.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

const REQUEST_TIMEOUT_MS = 15_000;
const UPLOAD_TIMEOUT_MS = 60_000;

export class ApiError extends Error {
  readonly status: number;
  readonly code?: string;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

/** Thrown when the server cannot be reached at all. */
export class NetworkError extends Error {
  constructor(message = "network request failed") {
    super(message);
    this.name = "NetworkError";
  }
}

type ErrorBody = { error?: string; code?: string };

type SessionHooks = {
  /** Called when the refresh token is rejected: the user must sign in again. */
  onSessionExpired: () => void;
  onTokens: (tokens: AuthTokens) => void;
};

let hooks: SessionHooks = { onSessionExpired: () => undefined, onTokens: () => undefined };

export function setSessionHooks(next: SessionHooks): void {
  hooks = next;
}

let refreshInFlight: Promise<AuthTokens | null> | null = null;

/** Single-flight refresh: concurrent 401s share one /auth/refresh call. */
function refreshTokens(): Promise<AuthTokens | null> {
  if (!refreshInFlight) {
    refreshInFlight = doRefresh().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

async function doRefresh(): Promise<AuthTokens | null> {
  const current = await getTokens();
  if (!current?.refreshToken) {
    return null;
  }
  try {
    const response = await fetchWithTimeout(`${requireBaseUrl()}${endpoints.refresh}`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ refreshToken: current.refreshToken })
    });
    if (response.status === 401 || response.status === 400) {
      await clearTokens();
      hooks.onSessionExpired();
      return null;
    }
    if (!response.ok) {
      return null; // transient server problem: keep the session, fail this request
    }
    const body = (await response.json()) as { accessToken?: string; token?: string; refreshToken?: string };
    const accessToken = body.accessToken ?? body.token;
    if (!accessToken || !body.refreshToken) {
      return null;
    }
    const next = { accessToken, refreshToken: body.refreshToken };
    await saveTokens(next);
    hooks.onTokens(next);
    return next;
  } catch {
    return null; // offline: keep the session
  }
}

/** Returns a usable access token, refreshing it first when it is about to expire. */
export async function getFreshAccessToken(): Promise<string | null> {
  const tokens = await getTokens();
  if (!tokens) {
    return null;
  }
  if (isExpiringSoon(tokens.accessToken, Date.now())) {
    const next = await refreshTokens();
    return next?.accessToken ?? null;
  }
  return tokens.accessToken;
}

function requireBaseUrl(): string {
  if (!API_BASE_URL) {
    throw new NetworkError("API URL not configured: set EXPO_PUBLIC_API_URL for this build");
  }
  return API_BASE_URL;
}

async function fetchWithTimeout(url: string, init: RequestInit, timeoutMs = REQUEST_TIMEOUT_MS, outer?: AbortSignal): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  const onAbort = () => controller.abort();
  outer?.addEventListener("abort", onAbort, { once: true });
  try {
    return await fetch(url, { ...init, signal: controller.signal });
  } catch (error) {
    if (outer?.aborted) {
      throw error;
    }
    throw new NetworkError(error instanceof Error ? error.message : undefined);
  } finally {
    clearTimeout(timer);
    outer?.removeEventListener("abort", onAbort);
  }
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  query?: Record<string, string | number | boolean | undefined>;
  /** Skip the Authorization header and 401 handling (login, register, ...). */
  anonymous?: boolean;
  signal?: AbortSignal;
};

function buildPath(path: string, query?: RequestOptions["query"]): string {
  if (!query) {
    return path;
  }
  const params = Object.entries(query)
    .filter((entry): entry is [string, string | number | boolean] => entry[1] !== undefined)
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`);
  return params.length ? `${path}?${params.join("&")}` : path;
}

async function send(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  const headers: Record<string, string> = { Accept: "application/json" };
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form; // fetch sets the multipart boundary itself
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return fetchWithTimeout(
    `${requireBaseUrl()}${buildPath(path, options.query)}`,
    { method: options.method ?? "GET", headers, body },
    options.form ? UPLOAD_TIMEOUT_MS : REQUEST_TIMEOUT_MS,
    options.signal
  );
}

export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  let token: string | null = null;
  if (!options.anonymous) {
    token = await getFreshAccessToken();
  }

  let response = await send(path, options, token);

  if (response.status === 401 && !options.anonymous) {
    const refreshed = await refreshTokens();
    if (refreshed) {
      response = await send(path, options, refreshed.accessToken);
    }
  }

  if (!response.ok) {
    let body: ErrorBody = {};
    try {
      body = (await response.json()) as ErrorBody;
    } catch {
      // Non-JSON error page (proxy, gateway): fall through to the generic message.
    }
    throw new ApiError(response.status, body.error ?? `request failed (${response.status})`, body.code);
  }

  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
