import { Platform } from "react-native";
import { clearTokens, getTokens, saveTokens } from "./tokenStore";
import type { AuthSession } from "./types";

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return explicit.replace(/\/+$/, "");
  }
  if (__DEV__) {
    // Android emulators reach the host through 10.0.2.2. Physical phones need EXPO_PUBLIC_API_URL.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

/** Absolute URL for a signed media path returned by the API. */
export function mediaUrl(path: string): string {
  return /^https?:\/\//.test(path) ? path : `${API_BASE_URL}${path}`;
}

type SessionListener = (session: AuthSession | null) => void;
let sessionListener: SessionListener | null = null;

/** The auth provider registers here to learn about refreshed or expired sessions. */
export function onSessionChange(listener: SessionListener | null) {
  sessionListener = listener;
}

let refreshInFlight: Promise<boolean> | null = null;

async function refreshTokens(): Promise<boolean> {
  if (refreshInFlight) {
    return refreshInFlight;
  }
  refreshInFlight = (async () => {
    const tokens = await getTokens();
    if (!tokens?.refreshToken) {
      return false;
    }
    try {
      const session = await rawRequest<AuthSession>("/auth/refresh", {
        method: "POST",
        body: JSON.stringify({ refreshToken: tokens.refreshToken })
      });
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      sessionListener?.(session);
      return true;
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        await clearTokens();
        sessionListener?.(null);
      }
      return false;
    }
  })();
  try {
    return await refreshInFlight;
  } finally {
    refreshInFlight = null;
  }
}

async function rawRequest<T>(path: string, init: RequestInit & { token?: string | null; timeoutMs?: number }): Promise<T> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "not_configured", "EXPO_PUBLIC_API_URL n’est pas configurée pour ce build.");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), init.timeoutMs ?? 15_000);
  init.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (init.body && !(init.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (init.token) {
    headers.set("Authorization", `Bearer ${init.token}`);
  }

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, { ...init, headers, signal: controller.signal });
  } catch (error) {
    const aborted = (error as { name?: string }).name === "AbortError";
    throw new ApiError(0, aborted ? "timeout" : "network", aborted ? "La requête a expiré." : "Impossible de joindre le serveur.");
  } finally {
    clearTimeout(timeout);
  }

  const text = await response.text();
  let payload: unknown = null;
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = null;
    }
  }
  if (!response.ok) {
    const body = (payload ?? {}) as { error?: string; code?: string };
    throw new ApiError(response.status, body.code ?? "http_error", body.error ?? `Erreur ${response.status}`);
  }
  return payload as T;
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

/**
 * Single entry point for REST calls. Attaches the access token and, when it
 * has expired, transparently refreshes the session once and retries.
 */
export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, form, auth = true, signal, timeoutMs } = options;
  const init = () => ({
    method,
    signal,
    timeoutMs,
    body: form ?? (body === undefined ? undefined : JSON.stringify(body))
  });

  const tokens = auth ? await getTokens() : null;
  try {
    return await rawRequest<T>(path, { ...init(), token: tokens?.accessToken });
  } catch (error) {
    if (!auth || !(error instanceof ApiError) || error.status !== 401) {
      throw error;
    }
    const refreshed = await refreshTokens();
    if (!refreshed) {
      throw error;
    }
    const next = await getTokens();
    return rawRequest<T>(path, { ...init(), token: next?.accessToken });
  }
}
