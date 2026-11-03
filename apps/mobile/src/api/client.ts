import { API_URL } from "../config";
import { clearTokens, getTokens, storeTokens, type Tokens } from "../auth/session";

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
    public readonly code?: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
};

type ErrorPayload = { error?: string; code?: string };

const TIMEOUT_MS = 15_000;
let refreshing: Promise<boolean> | null = null;

async function rawFetch(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), TIMEOUT_MS);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  try {
    return await fetch(`${API_URL}${path}`, { method: options.method ?? "GET", headers, body, signal: controller.signal });
  } catch (error) {
    if ((error as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "Le serveur met trop de temps à répondre.", "timeout");
    }
    throw new ApiError(0, "Connexion impossible. Vérifiez votre réseau puis réessayez.", "network");
  } finally {
    clearTimeout(timeout);
  }
}

/** Rotates the refresh token once for all concurrent callers. */
async function refreshSession(): Promise<boolean> {
  const tokens = getTokens();
  if (!tokens) {
    return false;
  }
  refreshing ??= (async () => {
    try {
      const response = await rawFetch("/auth/refresh", { method: "POST", body: { refreshToken: tokens.refreshToken } }, null);
      if (!response.ok) {
        await clearTokens();
        return false;
      }
      const next = (await response.json()) as Tokens;
      await storeTokens({ accessToken: next.accessToken, refreshToken: next.refreshToken });
      return true;
    } catch {
      return false; // transient failure: keep the session, caller sees the error
    } finally {
      refreshing = null;
    }
  })();
  return refreshing;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let response = await rawFetch(path, options, useAuth ? (getTokens()?.accessToken ?? null) : null);
  if (response.status === 401 && useAuth && getTokens()) {
    if (await refreshSession()) {
      response = await rawFetch(path, options, getTokens()?.accessToken ?? null);
    }
  }
  if (!response.ok) {
    let payload: ErrorPayload = {};
    try {
      payload = (await response.json()) as ErrorPayload;
    } catch {
      payload = {};
    }
    throw new ApiError(response.status, payload.error ?? `Erreur ${response.status}`, payload.code);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  return error instanceof Error && error.message ? error.message : fallback;
}
