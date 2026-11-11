import { API_BASE_URL } from "./config";
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

export class NetworkError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "NetworkError";
  }
}

/** Connects the client to the session owner (AuthProvider) without a cycle. */
export type SessionBridge = {
  getAccessToken: () => string | null;
  refresh: () => Promise<string | null>;
};

let bridge: SessionBridge | null = null;
let refreshInFlight: Promise<string | null> | null = null;

export function setSessionBridge(next: SessionBridge | null) {
  bridge = next;
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

async function parseError(response: Response): Promise<ApiError> {
  let code = "http_error";
  let message = `Erreur ${response.status}`;
  try {
    const payload = (await response.json()) as { error?: string; code?: string };
    code = payload.code ?? code;
    message = payload.error ?? message;
  } catch {
    // Non-JSON error body: keep defaults.
  }
  return new ApiError(response.status, code, message);
}

async function send(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  if (!API_BASE_URL) {
    throw new NetworkError("L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
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
    return await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (error) {
    if ((error as { name?: string }).name === "AbortError") {
      throw new NetworkError("Le serveur met trop de temps à répondre. Réessaie.");
    }
    throw new NetworkError("Connexion impossible. Vérifie ton réseau.");
  } finally {
    clearTimeout(timeout);
  }
}

function refreshOnce(): Promise<string | null> {
  if (!bridge) {
    return Promise.resolve(null);
  }
  if (!refreshInFlight) {
    refreshInFlight = bridge.refresh().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

/**
 * Performs an API call. Authenticated calls transparently refresh an expired
 * access token once (single-flight across concurrent requests) and retry.
 */
export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let token = useAuth ? (bridge?.getAccessToken() ?? null) : null;
  let response = await send(path, options, token);

  if (response.status === 401 && useAuth && token) {
    token = await refreshOnce();
    if (token) {
      response = await send(path, options, token);
    }
  }

  if (!response.ok) {
    throw await parseError(response);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof ApiError || error instanceof NetworkError) {
    return error.message;
  }
  return fallback;
}

export type { AuthSession };
