import { Platform } from "react-native";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }
  if (__DEV__) {
    // Android emulators reach the host through 10.0.2.2. Physical devices
    // must set EXPO_PUBLIC_API_URL to the computer's LAN IP.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** Turns a relative API path (e.g. a signed photo URL) into an absolute URL. */
export function absoluteUrl(path: string): string {
  if (/^https?:\/\//.test(path)) {
    return path;
  }
  return `${API_BASE_URL}${path}`;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof ApiError || error instanceof Error) {
    return error.message || fallback;
  }
  return fallback;
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  auth?: boolean;
  timeoutMs?: number;
  signal?: AbortSignal;
};

type SessionPayload = {
  accessToken: string;
  refreshToken: string;
};

let sessionExpiredHandler: (() => void) | null = null;
let refreshInFlight: Promise<boolean> | null = null;

/** Called once the session can no longer be refreshed. */
export function onSessionExpired(handler: () => void) {
  sessionExpiredHandler = handler;
}

/**
 * Refreshes the access token once, even when many requests fail at the same
 * time (single flight). Resolves to false when the user must sign in again.
 */
export function refreshAccessToken(): Promise<boolean> {
  if (refreshInFlight) {
    return refreshInFlight;
  }
  const tokens = getTokens();
  if (!tokens?.refreshToken) {
    return Promise.resolve(false);
  }
  refreshInFlight = (async () => {
    try {
      const session = await rawRequest<SessionPayload>("/auth/refresh", {
        method: "POST",
        body: { refreshToken: tokens.refreshToken },
        auth: false
      });
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      return true;
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        await clearTokens();
        sessionExpiredHandler?.();
      }
      return false;
    } finally {
      refreshInFlight = null;
    }
  })();
  return refreshInFlight;
}

async function rawRequest<T>(path: string, options: RequestOptions): Promise<T> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "not_configured", "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  if (options.signal) {
    if (options.signal.aborted) {
      controller.abort();
    } else {
      options.signal.addEventListener("abort", () => controller.abort(), { once: true });
    }
  }

  const headers: Record<string, string> = { Accept: "application/json" };
  let body: BodyInit | undefined;
  if (options.body instanceof FormData) {
    body = options.body;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  if (options.auth !== false) {
    const token = getTokens()?.accessToken;
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
  }

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (error) {
    if ((error as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "timeout", "Le serveur met trop de temps à répondre. Réessaie.");
    }
    throw new ApiError(0, "network", "Connexion impossible. Vérifie ton réseau puis réessaie.");
  } finally {
    clearTimeout(timeout);
  }

  const text = await response.text();
  let data: unknown = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = null;
    }
  }

  if (!response.ok) {
    const payload = (data ?? {}) as { error?: string; code?: string };
    const message =
      payload.error ??
      (response.status === 429 ? "Trop de tentatives, patiente un instant." : `Erreur ${response.status}`);
    throw new ApiError(response.status, payload.code ?? `http_${response.status}`, message);
  }
  return data as T;
}

/** Authenticated request with transparent access-token refresh. */
export async function apiRequest<T = unknown>(path: string, options: RequestOptions = {}): Promise<T> {
  try {
    return await rawRequest<T>(path, options);
  } catch (error) {
    const canRetry =
      error instanceof ApiError &&
      error.status === 401 &&
      options.auth !== false &&
      (error.code === "invalid_token" || error.code === "missing_token");
    if (!canRetry) {
      throw error;
    }
    const refreshed = await refreshAccessToken();
    if (!refreshed) {
      throw new ApiError(401, "session_expired", "Ta session a expiré, reconnecte-toi.");
    }
    return rawRequest<T>(path, options);
  }
}
