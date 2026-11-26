import { API_BASE_URL } from "./config";
import { clearTokens, currentTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

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

export const NETWORK_ERROR_CODE = "network_error";
const REQUEST_TIMEOUT_MS = 15_000;

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  /** Skip the Authorization header and the refresh-on-401 logic. */
  anonymous?: boolean;
  signal?: AbortSignal;
};

type ErrorPayload = { error?: string; code?: string };

let refreshInFlight: Promise<boolean> | null = null;

/** Exchanges the refresh token once, even when many requests fail with 401 at the same time. */
export function refreshSession(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = currentTokens();
      if (!tokens?.refreshToken) {
        return false;
      }
      try {
        const next = await rawRequest<{ accessToken: string; refreshToken: string }>("/auth/refresh", {
          method: "POST",
          body: { refreshToken: tokens.refreshToken },
          anonymous: true
        });
        await saveTokens({ accessToken: next.accessToken, refreshToken: next.refreshToken });
        return true;
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) {
          await clearTokens();
        }
        return false;
      }
    })().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

function buildBody(body: unknown): { payload?: BodyInit; contentType?: string } {
  if (body === undefined) {
    return {};
  }
  if (typeof FormData !== "undefined" && body instanceof FormData) {
    return { payload: body };
  }
  return { payload: JSON.stringify(body), contentType: "application/json" };
}

async function rawRequest<T>(path: string, options: RequestOptions, tokens?: AuthTokens | null): Promise<T> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "config_error", "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const { payload, contentType } = buildBody(options.body);
  const headers: Record<string, string> = { Accept: "application/json" };
  if (contentType) {
    headers["Content-Type"] = contentType;
  }
  if (!options.anonymous && tokens?.accessToken) {
    headers.Authorization = `Bearer ${tokens.accessToken}`;
  }

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body: payload,
      signal: controller.signal
    });
  } catch {
    throw new ApiError(0, NETWORK_ERROR_CODE, "Connexion impossible. Vérifiez votre réseau et réessayez.");
  } finally {
    clearTimeout(timeout);
  }

  const text = await response.text();
  if (!response.ok) {
    let data: ErrorPayload = {};
    try {
      data = text ? (JSON.parse(text) as ErrorPayload) : {};
    } catch {
      data = {};
    }
    throw new ApiError(
      response.status,
      data.code ?? `http_${response.status}`,
      data.error ?? "Une erreur est survenue. Réessayez dans un instant."
    );
  }
  return (text ? JSON.parse(text) : undefined) as T;
}

/** Authenticated request with transparent access-token refresh. */
export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  if (options.anonymous) {
    return rawRequest<T>(path, options);
  }
  try {
    return await rawRequest<T>(path, options, currentTokens());
  } catch (error) {
    if (error instanceof ApiError && error.status === 401 && currentTokens()?.refreshToken) {
      const refreshed = await refreshSession();
      if (refreshed) {
        return rawRequest<T>(path, options, currentTokens());
      }
    }
    throw error;
  }
}

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  return fallback;
}
