import { Platform } from "react-native";
import { getTokens, setTokens } from "./session";

// Single network gateway of the app: UI code never calls fetch directly.

function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return explicit.replace(/\/+$/, "");
  if (__DEV__) return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  return "";
}

export const API_BASE_URL = resolveBaseUrl();

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

export function isApiError(error: unknown, code?: string): error is ApiError {
  return error instanceof ApiError && (code === undefined || error.code === code);
}

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof ApiError) return error.message;
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

/** Turns a relative signed media path returned by the API into an absolute URL. */
export function mediaUrl(path: string): string {
  return /^https?:\/\//.test(path) ? path : `${API_BASE_URL}${path}`;
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  timeoutMs?: number;
  signal?: AbortSignal;
};

type AuthPayload = { accessToken: string; refreshToken: string };

let refreshInFlight: Promise<boolean> | null = null;
let sessionExpiredHandler: (() => void) | null = null;

export function onSessionExpired(handler: () => void): void {
  sessionExpiredHandler = handler;
}

/** Rotates the refresh token once even if several requests fail with 401 together. */
export function refreshSession(): Promise<boolean> {
  if (refreshInFlight) return refreshInFlight;
  const current = getTokens();
  if (!current?.refreshToken) return Promise.resolve(false);
  refreshInFlight = (async () => {
    try {
      const res = await rawRequest<AuthPayload>("/auth/refresh", {
        method: "POST",
        body: { refreshToken: current.refreshToken },
        auth: false
      });
      await setTokens({ accessToken: res.accessToken, refreshToken: res.refreshToken });
      return true;
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        await setTokens(null);
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
    throw new ApiError(0, "config", "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  const token = getTokens()?.accessToken;
  if (options.auth !== false && token) {
    headers.Authorization = `Bearer ${token}`;
  }

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, { method: options.method ?? "GET", headers, body, signal: controller.signal });
  } catch (error) {
    if ((error as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "timeout", "Le serveur met trop de temps à répondre. Réessaie.");
    }
    throw new ApiError(0, "network", "Connexion impossible. Vérifie ton réseau.");
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
    throw new ApiError(response.status, payload.code ?? `http_${response.status}`, payload.error ?? `Erreur ${response.status}`);
  }
  return data as T;
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  try {
    return await rawRequest<T>(path, options);
  } catch (error) {
    if (options.auth !== false && error instanceof ApiError && error.status === 401 && getTokens()?.refreshToken) {
      const refreshed = await refreshSession();
      if (refreshed) return rawRequest<T>(path, options);
    }
    throw error;
  }
}
