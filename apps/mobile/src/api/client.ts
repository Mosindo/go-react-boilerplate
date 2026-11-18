import { Platform } from "react-native";
import { clearTokens, currentTokens, loadTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }
  if (__DEV__) {
    // Android emulators reach the host machine through 10.0.2.2. Physical
    // devices always need EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** Photo URLs are returned relative (and signed) by the API. */
export function resolveMediaUrl(url: string): string {
  return /^https?:\/\//.test(url) ? url : `${API_BASE_URL}${url}`;
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

type SessionListener = {
  onTokens: (tokens: AuthTokens) => void;
  onExpired: () => void;
};

let sessionListener: SessionListener | null = null;

/** The auth provider registers here to follow silent refreshes and expiry. */
export function setSessionListener(listener: SessionListener | null) {
  sessionListener = listener;
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

type AuthPayload = { accessToken: string; refreshToken: string };

let refreshing: Promise<boolean> | null = null;

/** Single-flight refresh: concurrent 401s share one refresh request. */
async function refreshSession(): Promise<boolean> {
  if (!refreshing) {
    refreshing = (async () => {
      const tokens = currentTokens() ?? (await loadTokens());
      if (!tokens?.refreshToken) {
        return false;
      }
      try {
        const next = await rawRequest<AuthPayload>("/auth/refresh", {
          method: "POST",
          body: { refreshToken: tokens.refreshToken },
          auth: false
        });
        const saved = { accessToken: next.accessToken, refreshToken: next.refreshToken };
        await saveTokens(saved);
        sessionListener?.onTokens(saved);
        return true;
      } catch (error) {
        if (error instanceof ApiError) {
          await clearTokens();
          sessionListener?.onExpired();
        }
        return false;
      }
    })().finally(() => {
      refreshing = null;
    });
  }
  return refreshing;
}

async function rawRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  if (!API_BASE_URL) {
    throw new NetworkError("L’adresse de l’API n’est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (options.auth !== false) {
    const tokens = currentTokens() ?? (await loadTokens());
    if (tokens?.accessToken) {
      headers.Authorization = `Bearer ${tokens.accessToken}`;
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
    if ((error as { name?: string }).name === "AbortError" && !options.signal?.aborted) {
      throw new NetworkError("Le serveur met trop de temps à répondre. Réessayez.");
    }
    throw new NetworkError("Connexion impossible. Vérifiez votre réseau.");
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
    throw new ApiError(response.status, data.code ?? "unknown", data.error ?? `Erreur ${response.status}`);
  }
  return payload as T;
}

/** Authenticated request with one transparent retry after a token refresh. */
export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  try {
    return await rawRequest<T>(path, options);
  } catch (error) {
    if (error instanceof ApiError && error.status === 401 && options.auth !== false) {
      if (await refreshSession()) {
        return rawRequest<T>(path, options);
      }
    }
    throw error;
  }
}

/** Builds the WebSocket URL for the realtime endpoint. */
export function realtimeUrl(ticket: string): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/realtime/ws?ticket=${encodeURIComponent(ticket)}`;
}
