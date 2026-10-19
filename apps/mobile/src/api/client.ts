import { Platform } from "react-native";
import type { AuthSession } from "./types";

export type Tokens = { accessToken: string; refreshToken: string };

function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return explicit.replace(/\/+$/, "");
  if (__DEV__) return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  return "";
}

export const API_BASE_URL = resolveBaseUrl();

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string | undefined,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export class NetworkError extends Error {
  constructor(message = "network error") {
    super(message);
    this.name = "NetworkError";
  }
}

// --- session -----------------------------------------------------------------
// The API layer owns the current tokens so every request (and the WebSocket)
// shares one source of truth and one single-flight refresh.

type SessionListener = (tokens: Tokens | null) => void;

let tokens: Tokens | null = null;
let refreshInFlight: Promise<Tokens | null> | null = null;
const listeners = new Set<SessionListener>();

export function setTokens(next: Tokens | null): void {
  tokens = next;
  listeners.forEach((listener) => listener(next));
}

export function getTokens(): Tokens | null {
  return tokens;
}

export function subscribeTokens(listener: SessionListener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function jwtExpiryMs(token: string): number {
  try {
    const payload = token.split(".")[1];
    if (!payload) return 0;
    const json = JSON.parse(atob(payload.replace(/-/g, "+").replace(/_/g, "/"))) as { exp?: number };
    return (json.exp ?? 0) * 1000;
  } catch {
    return 0;
  }
}

async function refreshTokens(): Promise<Tokens | null> {
  if (!tokens) return null;
  const current = tokens;
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      try {
        const session = await request<AuthSession>("/auth/refresh", {
          method: "POST",
          body: { refreshToken: current.refreshToken },
          auth: false
        });
        const next = { accessToken: session.accessToken, refreshToken: session.refreshToken };
        setTokens(next);
        return next;
      } catch (error) {
        // Only a definitive rejection ends the session; a flaky network does not.
        if (error instanceof ApiError && (error.status === 401 || error.status === 400)) {
          setTokens(null);
        }
        return null;
      } finally {
        refreshInFlight = null;
      }
    })();
  }
  return refreshInFlight;
}

/** Returns an access token valid for at least 30s, refreshing it if needed. */
export async function getFreshAccessToken(): Promise<string | null> {
  if (!tokens) return null;
  if (jwtExpiryMs(tokens.accessToken) - Date.now() > 30_000) return tokens.accessToken;
  const next = await refreshTokens();
  return next?.accessToken ?? null;
}

// --- requests ----------------------------------------------------------------

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  query?: Record<string, string | number | undefined>;
  auth?: boolean;
  signal?: AbortSignal;
};

const TIMEOUT_MS = 15_000;

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  if (!API_BASE_URL) {
    throw new NetworkError("API base URL is not configured (set EXPO_PUBLIC_API_URL)");
  }
  const params = Object.entries(query ?? {})
    .filter((entry): entry is [string, string | number] => entry[1] !== undefined && entry[1] !== "")
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
    .join("&");
  return `${API_BASE_URL}${path}${params ? `?${params}` : ""}`;
}

async function send<T>(path: string, options: RequestOptions, accessToken: string | null): Promise<T> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), TIMEOUT_MS);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`;
  if (options.body !== undefined) headers["Content-Type"] = "application/json";

  let response: Response;
  try {
    response = await fetch(buildUrl(path, options.query), {
      method: options.method ?? "GET",
      headers,
      body: options.form ?? (options.body !== undefined ? JSON.stringify(options.body) : undefined),
      signal: controller.signal
    });
  } catch {
    throw new NetworkError();
  } finally {
    clearTimeout(timeout);
  }

  const text = await response.text();
  let data: unknown = undefined;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = undefined;
    }
  }

  if (!response.ok) {
    const payload = (data ?? {}) as { error?: string; code?: string };
    throw new ApiError(response.status, payload.code, payload.error ?? `Request failed (${response.status})`);
  }
  return data as T;
}

export async function request<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  const token = useAuth ? await getFreshAccessToken() : null;
  try {
    return await send<T>(path, options, token);
  } catch (error) {
    if (useAuth && error instanceof ApiError && error.status === 401 && tokens) {
      const next = await refreshTokens();
      if (next) return send<T>(path, options, next.accessToken);
    }
    throw error;
  }
}
