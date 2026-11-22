import { Platform } from "react-native";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import { errorMessages } from "./errors";

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return explicit.replace(/\/+$/, "");
  }
  if (__DEV__) {
    // Android emulators reach the host through 10.0.2.2.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();
const REQUEST_TIMEOUT_MS = 15_000;

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

/** Turns a relative media path returned by the API into an absolute URL. */
export function mediaUrl(path: string): string {
  return /^https?:\/\//.test(path) ? path : `${API_BASE_URL}${path}`;
}

type SessionListener = () => void;
const expiredListeners = new Set<SessionListener>();

/** Called when the refresh token is rejected and the user must sign in again. */
export function onSessionExpired(listener: SessionListener): () => void {
  expiredListeners.add(listener);
  return () => {
    expiredListeners.delete(listener);
  };
}

type AuthPayload = {
  accessToken: string;
  refreshToken: string;
};

let refreshing: Promise<string | null> | null = null;

/** Rotates the refresh token once, even if many requests fail concurrently. */
export function refreshAccessToken(): Promise<string | null> {
  if (refreshing) {
    return refreshing;
  }
  refreshing = (async () => {
    const refreshToken = getTokens()?.refreshToken;
    if (!refreshToken) {
      return null;
    }
    try {
      const res = await rawFetch("/auth/refresh", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refreshToken })
      });
      if (!res.ok) {
        if (res.status === 401 || res.status === 400) {
          await clearTokens();
          expiredListeners.forEach((l) => l());
        }
        return null;
      }
      const payload = (await res.json()) as AuthPayload;
      await saveTokens({ accessToken: payload.accessToken, refreshToken: payload.refreshToken });
      return payload.accessToken;
    } catch {
      return null;
    }
  })().finally(() => {
    refreshing = null;
  });
  return refreshing;
}

function decodeExpiry(token: string): number | null {
  try {
    const part = token.split(".")[1];
    const json = atob(
      part
        .replace(/-/g, "+")
        .replace(/_/g, "/")
        .padEnd(Math.ceil(part.length / 4) * 4, "=")
    );
    const exp = (JSON.parse(json) as { exp?: number }).exp;
    return typeof exp === "number" ? exp * 1000 : null;
  } catch {
    return null;
  }
}

/** Returns an access token valid for at least 60 s, refreshing if needed. */
export async function getFreshAccessToken(): Promise<string | null> {
  const token = getTokens()?.accessToken;
  if (!token) {
    return null;
  }
  const exp = decodeExpiry(token);
  if (exp !== null && exp - Date.now() < 60_000) {
    return refreshAccessToken();
  }
  return token;
}

async function rawFetch(path: string, init: RequestInit): Promise<Response> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    return await fetch(`${API_BASE_URL}${path}`, { ...init, signal: controller.signal });
  } catch (err) {
    if ((err as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "Le serveur met trop de temps à répondre. Réessayez.", "timeout");
    }
    throw new ApiError(0, "Connexion impossible. Vérifiez votre réseau.", "network");
  } finally {
    clearTimeout(timer);
  }
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
};

async function parseError(res: Response): Promise<ApiError> {
  let message = `Erreur ${res.status}`;
  let code: string | undefined;
  try {
    const data = (await res.json()) as { error?: string; code?: string };
    message = data.error ?? message;
    code = data.code;
  } catch {
    // Non-JSON error body.
  }
  if (code && errorMessages[code]) {
    message = errorMessages[code];
  } else if (res.status === 429) {
    message = errorMessages.rate_limited;
  } else if (res.status >= 500) {
    message = "Le service est momentanément indisponible. Réessayez.";
  }
  return new ApiError(res.status, message, code);
}

/**
 * The single entry point for HTTP calls. Authenticated requests transparently
 * refresh an expired access token once and retry.
 */
export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, form, auth = true } = options;

  const send = async (token: string | null) => {
    const headers: Record<string, string> = { Accept: "application/json" };
    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
    }
    if (auth && token) {
      headers.Authorization = `Bearer ${token}`;
    }
    return rawFetch(path, {
      method,
      headers,
      body: form ?? (body !== undefined ? JSON.stringify(body) : undefined)
    });
  };

  let token = auth ? await getFreshAccessToken() : null;
  let res = await send(token);
  if (auth && res.status === 401 && getTokens()?.refreshToken) {
    token = await refreshAccessToken();
    if (token) {
      res = await send(token);
    }
  }
  if (!res.ok) {
    throw await parseError(res);
  }
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export function errorMessage(err: unknown, fallback = "Une erreur est survenue."): string {
  if (err instanceof ApiError || err instanceof Error) {
    return err.message || fallback;
  }
  return fallback;
}
