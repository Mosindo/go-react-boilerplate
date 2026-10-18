import { Platform } from "react-native";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import { showGlobalError } from "../shared/feedback";

export type ApiErrorPayload = {
  error?: string;
  code?: string;
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

const REQUEST_TIMEOUT_MS = 12_000;
const UPLOAD_TIMEOUT_MS = 45_000;
const PUBLIC_PATHS = ["/auth/login", "/auth/register", "/auth/refresh", "/auth/password/"];

export class ApiError extends Error {
  public status: number;
  public code?: string;
  public data: unknown;

  constructor(status: number, data: unknown, message?: string) {
    super(message ?? `API request failed (${status})`);
    this.name = "ApiError";
    this.status = status;
    this.data = data;
    this.code = (data as ApiErrorPayload | null)?.code;
  }
}

type SessionListener = () => void;
const sessionExpiredListeners = new Set<SessionListener>();

/** Called when the refresh token is rejected: the user must sign in again. */
export function onSessionExpired(listener: SessionListener): () => void {
  sessionExpiredListeners.add(listener);
  return () => {
    sessionExpiredListeners.delete(listener);
  };
}

function networkHint(): string {
  if (!API_BASE_URL) {
    return "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).";
  }
  return "Impossible de joindre le serveur. Vérifiez votre connexion et réessayez.";
}

let refreshInFlight: Promise<boolean> | null = null;

/** Single-flight refresh: concurrent 401s share one /auth/refresh call. */
async function refreshAccessToken(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = await getTokens();
      if (!tokens?.refreshToken) {
        return false;
      }
      try {
        const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ refreshToken: tokens.refreshToken })
        });
        if (!response.ok) {
          return false;
        }
        const body = (await response.json()) as { accessToken?: string; token?: string; refreshToken?: string };
        const accessToken = body.accessToken ?? body.token;
        if (!accessToken || !body.refreshToken) {
          return false;
        }
        await saveTokens({ accessToken, refreshToken: body.refreshToken });
        return true;
      } catch {
        // A network failure is not a rejected session: keep the tokens.
        return true;
      }
    })().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

async function parseError(response: Response): Promise<ApiError> {
  const contentType = response.headers.get("content-type") ?? "";
  let data: unknown = null;
  try {
    data = contentType.includes("application/json") ? await response.json() : await response.text();
  } catch {
    data = null;
  }
  const payload = data as ApiErrorPayload | null;
  const fallback = response.status === 429 ? "Trop de requêtes, patientez un instant." : `Erreur ${response.status}`;
  return new ApiError(response.status, data, payload?.error ?? fallback);
}

export type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
  /** Set for multipart uploads; the browser/runtime adds the boundary header. */
  formData?: FormData;
};

export async function apiRequest<T = unknown>(path: string, options: RequestOptions = {}, retried = false): Promise<T> {
  if (!API_BASE_URL) {
    const message = networkHint();
    showGlobalError(message);
    throw new Error(message);
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), options.formData ? UPLOAD_TIMEOUT_MS : REQUEST_TIMEOUT_MS);
  if (options.signal) {
    if (options.signal.aborted) {
      controller.abort();
    } else {
      options.signal.addEventListener("abort", () => controller.abort(), { once: true });
    }
  }

  const isPublic = PUBLIC_PATHS.some((p) => path.startsWith(p));
  const tokens = isPublic ? null : await getTokens();
  const headers = new Headers(options.headers);
  headers.set("Accept", "application/json");
  if (tokens?.accessToken) {
    headers.set("Authorization", `Bearer ${tokens.accessToken}`);
  }
  let body: BodyInit | undefined;
  if (options.formData) {
    body = options.formData;
  } else if (options.body !== undefined) {
    headers.set("Content-Type", "application/json");
    body = JSON.stringify(options.body);
  }

  try {
    const response = await fetch(`${API_BASE_URL}${path}`, { ...options, headers, body, signal: controller.signal });

    if (response.status === 401 && tokens && !isPublic && !retried) {
      if (await refreshAccessToken()) {
        return await apiRequest<T>(path, options, true);
      }
      await clearTokens();
      sessionExpiredListeners.forEach((listener) => listener());
    }
    if (!response.ok) {
      const error = await parseError(response);
      if (response.status >= 500) {
        showGlobalError("Le serveur rencontre un problème. Réessayez dans un instant.");
      }
      throw error;
    }
    const text = await response.text();
    return (text ? JSON.parse(text) : undefined) as T;
  } catch (err) {
    if (err instanceof ApiError) {
      throw err;
    }
    if ((err as { name?: string }).name === "AbortError") {
      showGlobalError("La requête a expiré. Réessayez.");
      throw new Error("La requête a expiré.");
    }
    if (err instanceof TypeError) {
      showGlobalError(networkHint());
      throw new Error(networkHint());
    }
    throw err;
  } finally {
    clearTimeout(timeoutId);
  }
}

/** Human readable message for any thrown value, for use in forms. */
export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  return error instanceof Error && error.message ? error.message : fallback;
}
