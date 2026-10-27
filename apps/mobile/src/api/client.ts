import Constants from "expo-constants";
import { Platform } from "react-native";
import { clearTokens, currentTokens, loadTokens, saveTokens } from "./tokenStore";
import type { AuthSession } from "./types";

const API_PORT = 18080;

function normalize(url: string): string {
  return url.replace(/\/+$/, "");
}

/**
 * Base URL resolution order:
 * 1. EXPO_PUBLIC_API_URL (required on physical devices: use your LAN IP, never localhost)
 * 2. the Expo dev server host (works for physical devices on the same network)
 * 3. emulator defaults
 */
function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return normalize(explicit);
  const hostUri = Constants.expoConfig?.hostUri;
  if (hostUri) {
    const host = hostUri.split(":")[0];
    if (host) return `http://${host}:${API_PORT}`;
  }
  return Platform.OS === "android" ? `http://10.0.2.2:${API_PORT}` : `http://localhost:${API_PORT}`;
}

export const API_BASE_URL = resolveBaseUrl();

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

type Listener = () => void;
const sessionExpiredListeners = new Set<Listener>();

/** Subscribe to "the refresh token is no longer valid": the UI must sign out. */
export function onSessionExpired(listener: Listener): () => void {
  sessionExpiredListeners.add(listener);
  return () => {
    sessionExpiredListeners.delete(listener);
  };
}

let refreshing: Promise<boolean> | null = null;

/** Single-flight token refresh: concurrent 401s share one request. */
function refreshTokens(): Promise<boolean> {
  if (!refreshing) {
    refreshing = (async () => {
      const tokens = (await loadTokens()) ?? currentTokens();
      if (!tokens) return false;
      try {
        const res = await fetch(`${API_BASE_URL}/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refreshToken: tokens.refreshToken })
        });
        if (!res.ok) {
          if (res.status === 401) {
            await clearTokens();
            sessionExpiredListeners.forEach((l) => l());
          }
          return false;
        }
        const session = (await res.json()) as AuthSession;
        await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
        return true;
      } catch {
        return false;
      } finally {
        refreshing = null;
      }
    })();
  }
  return refreshing;
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  signal?: AbortSignal;
  /** Skip the Authorization header and the refresh-on-401 behaviour. */
  anonymous?: boolean;
};

const TIMEOUT_MS = 15_000;

async function send(path: string, options: RequestOptions): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), TIMEOUT_MS);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (options.body !== undefined) headers["Content-Type"] = "application/json";
  const token = options.anonymous ? null : currentTokens()?.accessToken;
  if (token) headers.Authorization = `Bearer ${token}`;

  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body: options.form ?? (options.body !== undefined ? JSON.stringify(options.body) : undefined),
      signal: controller.signal
    });
  } finally {
    clearTimeout(timeout);
  }
}

export async function request<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  let res: Response;
  try {
    res = await send(path, options);
    if (res.status === 401 && !options.anonymous && (await refreshTokens())) {
      res = await send(path, options);
    }
  } catch (err) {
    if ((err as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "The request timed out. Please try again.");
    }
    throw new ApiError(0, "Cannot reach the server. Check your connection and try again.");
  }

  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const data = (await res.json()) as { error?: string };
      if (data.error) message = data.error;
    } catch {
      // Non-JSON error body: keep the generic message.
    }
    throw new ApiError(res.status, message);
  }
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

/** Absolute URL for a server-relative path such as a photo URL. */
export function absoluteUrl(path: string): string {
  return `${API_BASE_URL}${path}`;
}

/** Headers needed to fetch protected media (photos). */
export function mediaHeaders(): Record<string, string> {
  const token = currentTokens()?.accessToken;
  return token ? { Authorization: `Bearer ${token}` } : {};
}

export function websocketUrl(): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws`;
}
