import { API_BASE_URL } from "./config";
import { clearTokens, currentTokens, saveTokens } from "./session";
import type { AuthSession } from "./types";

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

export function isApiError(error: unknown, code?: string): error is ApiError {
  return error instanceof ApiError && (code === undefined || error.code === code);
}

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof ApiError || error instanceof Error) {
    return error.message || fallback;
  }
  return fallback;
}

type Listener = () => void;
const sessionExpiredListeners = new Set<Listener>();

/** Called when the refresh token is rejected: the user must sign in again. */
export function onSessionExpired(listener: Listener): () => void {
  sessionExpiredListeners.add(listener);
  return () => sessionExpiredListeners.delete(listener);
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

let refreshing: Promise<boolean> | null = null;

/** Rotates the refresh token once, even when many requests fail together. */
export function refreshSession(): Promise<boolean> {
  if (refreshing) {
    return refreshing;
  }
  refreshing = (async () => {
    const tokens = currentTokens();
    if (!tokens?.refreshToken) {
      return false;
    }
    try {
      const session = await send<AuthSession>("/auth/refresh", {
        method: "POST",
        body: { refreshToken: tokens.refreshToken },
        auth: false
      });
      await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
      return true;
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        await clearTokens();
        sessionExpiredListeners.forEach((listener) => listener());
      }
      return false;
    }
  })().finally(() => {
    refreshing = null;
  });
  return refreshing;
}

async function send<T>(path: string, options: RequestOptions): Promise<T> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "not_configured", "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (options.auth !== false) {
    const token = currentTokens()?.accessToken;
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
  }
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
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
      throw new ApiError(0, "timeout", "Le serveur met trop de temps à répondre. Réessayez.");
    }
    throw new ApiError(0, "network", "Connexion impossible. Vérifiez votre réseau et réessayez.");
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
    const data = (payload ?? {}) as { error?: string; code?: string };
    throw new ApiError(response.status, data.code ?? "http_error", data.error ?? `Erreur ${response.status}`);
  }
  return payload as T;
}

/** Authenticated request that transparently refreshes an expired access token. */
export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  try {
    return await send<T>(path, options);
  } catch (error) {
    if (options.auth !== false && error instanceof ApiError && error.status === 401 && error.code === "invalid_token") {
      if (await refreshSession()) {
        return send<T>(path, options);
      }
    }
    throw error;
  }
}
