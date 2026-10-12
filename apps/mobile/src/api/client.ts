import { Platform } from "react-native";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";

export type ApiErrorPayload = { error?: string };

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

/**
 * The API base URL comes from EXPO_PUBLIC_API_URL. Physical phones cannot reach
 * "localhost": use the LAN IP of the machine running the API.
 */
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
  public readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export function isApiError(error: unknown, status?: number): error is ApiError {
  return error instanceof ApiError && (status === undefined || error.status === status);
}

/** Turns a relative media path returned by the API into an absolute URL. */
export function absoluteUrl(path: string): string {
  return path.startsWith("http") ? path : `${API_BASE_URL}${path}`;
}

export function wsUrl(): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws`;
}

let onSessionLost: (() => void) | null = null;

/** The auth layer registers a callback invoked when the session cannot be refreshed. */
export function setSessionLostHandler(handler: (() => void) | null): void {
  onSessionLost = handler;
}

let refreshInFlight: Promise<boolean> | null = null;

/** Single-flight token refresh: concurrent 401s share one refresh request. */
async function refreshTokens(): Promise<boolean> {
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
          if (response.status === 401 || response.status === 400) {
            await clearTokens();
          }
          return false;
        }
        const body = (await response.json()) as { accessToken?: string; token?: string; refreshToken: string };
        const accessToken = body.accessToken ?? body.token;
        if (!accessToken) {
          return false;
        }
        await saveTokens({ accessToken, refreshToken: body.refreshToken });
        return true;
      } catch {
        return false;
      }
    })().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

function networkHint(): string {
  if (!API_BASE_URL) {
    return "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).";
  }
  return "Connexion impossible. Vérifiez votre réseau et réessayez.";
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  formData?: FormData;
  signal?: AbortSignal;
  /** Skip the Authorization header and the refresh-on-401 logic (login, register...). */
  anonymous?: boolean;
};

async function send(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.formData ? 60_000 : 15_000);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (token && !options.anonymous) {
    headers.Authorization = `Bearer ${token}`;
  }
  let body: BodyInit | undefined;
  if (options.formData) {
    body = options.formData;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  try {
    return await fetch(`${API_BASE_URL}${path}`, { method: options.method ?? "GET", headers, body, signal: controller.signal });
  } catch (error) {
    if ((error as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "La requête a expiré. Réessayez.");
    }
    throw new ApiError(0, networkHint());
  } finally {
    clearTimeout(timeout);
  }
}

export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  let token = options.anonymous ? null : ((await getTokens())?.accessToken ?? null);
  let response = await send(path, options, token);

  if (response.status === 401 && !options.anonymous) {
    if (await refreshTokens()) {
      token = (await getTokens())?.accessToken ?? null;
      response = await send(path, options, token);
    }
    if (response.status === 401) {
      onSessionLost?.();
    }
  }

  if (!response.ok) {
    let message = `Erreur ${response.status}`;
    try {
      const payload = (await response.json()) as ApiErrorPayload;
      if (payload.error) {
        message = payload.error;
      }
    } catch {
      // keep the generic message
    }
    throw new ApiError(response.status, message);
  }

  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

/** Human-readable message for any thrown value. */
export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  return error instanceof Error && error.message ? error.message : fallback;
}
