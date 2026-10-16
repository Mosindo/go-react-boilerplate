import { Platform } from "react-native";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import { showGlobalError } from "../shared/feedback";
import { singleFlight } from "./singleFlight";

export type ApiErrorPayload = {
  error?: string;
};

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
  return __DEV__ ? developmentFallbackBaseUrl() : "";
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

export type RequestOptions = Omit<RequestInit, "signal"> & {
  signal?: AbortSignal | null;
  /** Override the default 10 s timeout (uploads need longer). */
  timeoutMs?: number;
  /** Auth endpoints must not trigger a token refresh on 401. */
  skipAuthRefresh?: boolean;
  /** Skip the global error banner; the caller shows its own message. */
  silent?: boolean;
};

const sessionExpiredListeners = new Set<() => void>();

/** Notifies the app when the refresh token is rejected, so it can return to the sign-in screen. */
export function onSessionExpired(listener: () => void): () => void {
  sessionExpiredListeners.add(listener);
  return () => {
    sessionExpiredListeners.delete(listener);
  };
}

function requireApiBaseUrl(): string {
  if (API_BASE_URL) {
    return API_BASE_URL;
  }
  const message = "L'adresse de l'API n'est pas configurée pour cette version (EXPO_PUBLIC_API_URL).";
  showGlobalError(message);
  throw new Error(message);
}

export function absoluteUrl(path: string): string {
  return /^https?:\/\//.test(path) ? path : `${API_BASE_URL}${path}`;
}

function networkHint(): string {
  if (process.env.EXPO_PUBLIC_API_URL) {
    return "Connexion impossible. Vérifiez votre réseau puis réessayez.";
  }
  if (!__DEV__) {
    return "L'adresse de l'API n'est pas configurée pour cette version.";
  }
  if (Platform.OS === "android") {
    return "Connexion impossible. Un émulateur Android utilise 10.0.2.2 ; un téléphone réel nécessite EXPO_PUBLIC_API_URL avec l'IP de votre ordinateur.";
  }
  return "Connexion impossible. Un téléphone réel nécessite EXPO_PUBLIC_API_URL avec l'IP de votre ordinateur.";
}

async function readPayload(response: Response): Promise<unknown> {
  const contentType = response.headers.get("content-type") ?? "";
  try {
    return contentType.includes("application/json") ? await response.json() : await response.text();
  } catch {
    return null;
  }
}

async function send(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), options.timeoutMs ?? 10_000);
  if (options.signal) {
    if (options.signal.aborted) {
      controller.abort();
    } else {
      options.signal.addEventListener("abort", () => controller.abort(), { once: true });
    }
  }

  const headers = new Headers(options.headers);
  headers.set("Accept", "application/json");
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const hasBody = options.body !== undefined && options.body !== null;
  if (hasBody && !(options.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  try {
    return await fetch(`${requireApiBaseUrl()}${path}`, { ...options, headers, signal: controller.signal });
  } finally {
    clearTimeout(timeoutId);
  }
}

/**
 * Refresh tokens rotate on every use, so concurrent 401s must share a single refresh.
 * Returns the new access token, or null when the session is definitively over.
 * Network failures propagate: being offline is not a reason to sign the user out.
 */
const refreshAccessToken = singleFlight(async (): Promise<string | null> => {
  const tokens = await getTokens();
  if (!tokens?.refreshToken) {
    return null;
  }
  const response = await send("/auth/refresh", { method: "POST", body: JSON.stringify({ refreshToken: tokens.refreshToken }) }, null);
  if (!response.ok) {
    if (response.status === 400 || response.status === 401) {
      return null;
    }
    throw new ApiError(response.status, await readPayload(response));
  }
  const data = (await response.json()) as { accessToken?: string; token?: string; refreshToken?: string };
  const accessToken = data.accessToken ?? data.token;
  if (!accessToken || !data.refreshToken) {
    return null;
  }
  await saveTokens({ accessToken, refreshToken: data.refreshToken });
  return accessToken;
});

async function endSession(): Promise<void> {
  await clearTokens();
  sessionExpiredListeners.forEach((listener) => listener());
}

export async function apiRequest<T = unknown>(path: string, options: RequestOptions = {}): Promise<T> {
  try {
    const tokens = await getTokens();
    let response = await send(path, options, tokens?.accessToken ?? null);

    if (response.status === 401 && tokens && !options.skipAuthRefresh) {
      const refreshed = await refreshAccessToken();
      if (!refreshed) {
        await endSession();
        throw new ApiError(401, await readPayload(response), "Votre session a expiré. Reconnectez-vous.");
      }
      response = await send(path, options, refreshed);
    }

    if (!response.ok) {
      const data = await readPayload(response);
      const payload = data as ApiErrorPayload | null;
      const message = payload?.error ?? `API request failed (${response.status})`;
      if (response.status >= 500 && !options.silent) {
        showGlobalError("Le service est momentanément indisponible. Réessayez plus tard.");
      }
      throw new ApiError(response.status, data, message);
    }

    const text = await response.text();
    return (text ? JSON.parse(text) : undefined) as T;
  } catch (err) {
    if ((err as { name?: string }).name === "AbortError") {
      if (!options.silent) {
        showGlobalError("La requête a expiré. Réessayez.");
      }
      throw new Error("La requête a expiré.");
    }
    if (err instanceof TypeError && !options.silent) {
      showGlobalError(networkHint());
    }
    throw err;
  }
}
