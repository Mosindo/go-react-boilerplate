import { Platform } from "react-native";

/**
 * Single HTTP entry point of the app. Screens never call fetch directly: they
 * go through the typed modules in src/api, which use apiRequest.
 */

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

/** Signed media paths are returned relative by the API. */
export function mediaUrl(path: string): string {
  return /^https?:\/\//.test(path) ? path : `${API_BASE_URL}${path}`;
}

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

export class NetworkError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "NetworkError";
  }
}

type SessionBridge = {
  getAccessToken: () => string | null;
  /** Refreshes the session; resolves to the new access token or null. */
  refresh: () => Promise<string | null>;
};

let bridge: SessionBridge = {
  getAccessToken: () => null,
  refresh: async () => null
};

/** Called once by the auth provider to plug token storage into the client. */
export function configureSession(next: SessionBridge) {
  bridge = next;
}

let refreshInFlight: Promise<string | null> | null = null;

function refreshOnce(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = bridge.refresh().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  timeoutMs?: number;
  signal?: AbortSignal;
};

async function send(path: string, opts: RequestOptions, token: string | null): Promise<Response> {
  if (!API_BASE_URL) {
    throw new NetworkError("L'URL de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), opts.timeoutMs ?? 15_000);
  opts.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  let body: BodyInit | undefined;
  if (opts.form) {
    body = opts.form;
  } else if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(opts.body);
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  try {
    return await fetch(`${API_BASE_URL}${path}`, { method: opts.method ?? "GET", headers, body, signal: controller.signal });
  } catch (err) {
    if ((err as { name?: string }).name === "AbortError") {
      throw new NetworkError("Le serveur met trop de temps à répondre.");
    }
    throw new NetworkError("Connexion impossible. Vérifie ton réseau.");
  } finally {
    clearTimeout(timer);
  }
}

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

export async function apiRequest<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const useAuth = opts.auth !== false;
  let response = await send(path, opts, useAuth ? bridge.getAccessToken() : null);

  // Access tokens are short-lived: refresh once transparently and retry.
  if (response.status === 401 && useAuth && bridge.getAccessToken()) {
    const nextToken = await refreshOnce();
    if (nextToken) {
      response = await send(path, opts, nextToken);
    }
  }

  if (!response.ok) {
    throw await parseError(response);
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
