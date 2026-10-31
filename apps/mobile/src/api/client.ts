import { Platform } from "react-native";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import { showGlobalError } from "../shared/feedback";

export type ApiErrorPayload = { error?: string };

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }
  if (__DEV__) {
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export class ApiError extends Error {
  public status: number;
  public data: unknown;

  constructor(status: number, data: unknown, message?: string) {
    super(message ?? `API request failed (${status})`);
    this.name = "ApiError";
    this.status = status;
    this.data = data;
  }
}

export class NetworkError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "NetworkError";
  }
}

function networkHint(): string {
  if (!API_BASE_URL) {
    return "L'URL de l'API n'est pas configurée. Définissez EXPO_PUBLIC_API_URL.";
  }
  if (Platform.OS === "android" || Platform.OS === "ios") {
    return "Connexion impossible. Sur un téléphone physique, EXPO_PUBLIC_API_URL doit pointer vers l'IP de votre machine (pas localhost).";
  }
  return "Connexion impossible. Vérifiez votre réseau puis réessayez.";
}

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
  /** Skip the global error banner (the caller renders its own message). */
  silent?: boolean;
  /** Do not attempt a token refresh on 401 (used by auth endpoints). */
  noAuth?: boolean;
  timeoutMs?: number;
};

type RefreshResponse = { accessToken: string; refreshToken: string };

let refreshInFlight: Promise<string | null> | null = null;
let onSessionExpired: (() => void) | null = null;

/** The auth provider registers a callback to be told when refresh fails. */
export function setSessionExpiredHandler(handler: (() => void) | null): void {
  onSessionExpired = handler;
}

function tokenExpiresSoon(token: string, marginSeconds = 60): boolean {
  try {
    const payload = token.split(".")[1];
    if (!payload) return true;
    const json = JSON.parse(decodeBase64Url(payload)) as { exp?: number };
    return typeof json.exp !== "number" || json.exp * 1000 - Date.now() < marginSeconds * 1000;
  } catch {
    return true;
  }
}

const B64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

function decodeBase64Url(input: string): string {
  const clean = input.replace(/-/g, "+").replace(/_/g, "/").replace(/=+$/, "");
  let bits = 0;
  let acc = 0;
  let out = "";
  for (const ch of clean) {
    const v = B64.indexOf(ch);
    if (v < 0) continue;
    acc = (acc << 6) | v;
    bits += 6;
    if (bits >= 8) {
      bits -= 8;
      out += String.fromCharCode((acc >> bits) & 0xff);
    }
  }
  return decodeURIComponent(escape(out));
}

/**
 * Returns a valid access token, rotating it first when it is about to expire.
 * Photos are fetched with this token, so it is also refreshed in the background.
 */
export async function ensureFreshAccessToken(): Promise<string | null> {
  const tokens = await getTokens();
  if (!tokens) return null;
  if (!tokenExpiresSoon(tokens.accessToken)) return tokens.accessToken;
  return (await refreshAccessToken()) ?? tokens.accessToken;
}

/** Single-flight refresh so concurrent 401s trigger only one rotation. */
async function refreshAccessToken(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = await getTokens();
      if (!tokens?.refreshToken) {
        return null;
      }
      try {
        const res = await fetch(`${API_BASE_URL}/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ refreshToken: tokens.refreshToken })
        });
        if (!res.ok) {
          await clearTokens();
          onSessionExpired?.();
          return null;
        }
        const next = (await res.json()) as RefreshResponse;
        await saveTokens({ accessToken: next.accessToken, refreshToken: next.refreshToken });
        return next.accessToken;
      } catch {
        return null; // offline: keep the session, the caller will see a network error
      } finally {
        refreshInFlight = null;
      }
    })();
  }
  return refreshInFlight;
}

async function parseError(response: Response): Promise<ApiError> {
  let data: unknown = null;
  try {
    const type = response.headers.get("content-type") ?? "";
    data = type.includes("application/json") ? await response.json() : await response.text();
  } catch {
    data = null;
  }
  const message = (data as ApiErrorPayload | null)?.error ?? `Erreur serveur (${response.status})`;
  return new ApiError(response.status, data, message);
}

export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, silent, noAuth, timeoutMs = 10_000, headers: extraHeaders, ...init } = options;
  if (!API_BASE_URL) {
    const message = networkHint();
    if (!silent) showGlobalError(message);
    throw new NetworkError(message);
  }

  const isForm = typeof FormData !== "undefined" && body instanceof FormData;

  const attempt = async (token: string | null): Promise<Response> => {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);
    try {
      const headers = new Headers(extraHeaders);
      headers.set("Accept", "application/json");
      if (body !== undefined && !isForm) headers.set("Content-Type", "application/json");
      if (token) headers.set("Authorization", `Bearer ${token}`);
      return await fetch(`${API_BASE_URL}${path}`, {
        ...init,
        headers,
        body: body === undefined ? undefined : isForm ? (body as FormData) : JSON.stringify(body),
        signal: controller.signal
      });
    } finally {
      clearTimeout(timer);
    }
  };

  let response: Response;
  try {
    const tokens = noAuth ? null : await getTokens();
    const token = noAuth ? null : await ensureFreshAccessToken();
    response = await attempt(token);
    if (response.status === 401 && !noAuth && tokens?.refreshToken) {
      const fresh = await refreshAccessToken();
      if (fresh) {
        response = await attempt(fresh);
      }
    }
  } catch (err) {
    const aborted = (err as { name?: string }).name === "AbortError";
    const message = aborted ? "La requête a expiré. Réessayez." : networkHint();
    if (!silent) showGlobalError(message);
    throw new NetworkError(message);
  }

  if (!response.ok) {
    const error = await parseError(response);
    if (!silent && error.status >= 500) showGlobalError("Le serveur a rencontré un problème. Réessayez dans un instant.");
    throw error;
  }

  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

/** Image source for authenticated photo URLs (served by the API, never public). */
export function photoSource(url: string, accessToken: string | null) {
  return {
    uri: `${API_BASE_URL}${url}`,
    headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : undefined,
    cacheKey: url
  };
}

export function websocketUrl(): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws`;
}
