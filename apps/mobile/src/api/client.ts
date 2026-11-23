import { Platform } from "react-native";
import { showGlobalError } from "../shared/feedback/store";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }
  if (__DEV__) {
    // Android emulators reach the host through 10.0.2.2. Physical devices
    // need EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export function mediaUrl(path: string): string {
  return /^https?:\/\//.test(path) ? path : `${API_BASE_URL}${path}`;
}

export function realtimeUrl(): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/realtime`;
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

export function isApiError(error: unknown, code?: string): error is ApiError {
  return error instanceof ApiError && (code === undefined || error.code === code);
}

type SessionBridge = {
  getAccessToken: () => string | null;
  refresh: () => Promise<string | null>;
};

let session: SessionBridge = {
  getAccessToken: () => null,
  refresh: async () => null
};

/** Connects the HTTP client to the auth state (see useAuth). */
export function setSessionBridge(bridge: SessionBridge) {
  session = bridge;
}

let refreshing: Promise<string | null> | null = null;

function refreshOnce(): Promise<string | null> {
  if (!refreshing) {
    refreshing = session.refresh().finally(() => {
      refreshing = null;
    });
  }
  return refreshing;
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  formData?: FormData;
  auth?: boolean;
  timeoutMs?: number;
  signal?: AbortSignal;
};

async function send(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "not_configured", "API URL manquante : définis EXPO_PUBLIC_API_URL pour ce build.");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  let body: BodyInit | undefined;
  if (options.formData) {
    body = options.formData;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } finally {
    clearTimeout(timeout);
  }
}

async function toApiError(response: Response): Promise<ApiError> {
  let code = "http_error";
  let message = `Erreur ${response.status}`;
  try {
    const payload = (await response.json()) as { error?: string; code?: string };
    code = payload.code ?? code;
    message = payload.error ?? message;
  } catch {
    // Non-JSON body.
  }
  return new ApiError(response.status, code, message);
}

/**
 * Typed request helper. Every network access of the app goes through here:
 * it attaches the access token, refreshes it once on 401 and normalizes
 * errors into ApiError.
 */
export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let response: Response;
  try {
    response = await send(path, options, useAuth ? session.getAccessToken() : null);
    if (response.status === 401 && useAuth && session.getAccessToken()) {
      const next = await refreshOnce();
      if (next) {
        response = await send(path, options, next);
      }
    }
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    const aborted = (error as { name?: string }).name === "AbortError";
    if (!options.signal?.aborted) {
      showGlobalError(aborted ? "Le serveur met trop de temps à répondre." : "Connexion impossible. Vérifie ton réseau.");
    }
    throw new ApiError(0, aborted ? "timeout" : "network", aborted ? "Délai dépassé" : "Erreur réseau");
  }

  if (!response.ok) {
    const error = await toApiError(response);
    if (response.status >= 500) {
      showGlobalError("Un problème est survenu de notre côté. Réessaie dans un instant.");
    }
    throw error;
  }
  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
