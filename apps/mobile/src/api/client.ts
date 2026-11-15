import { Platform } from "react-native";
import { clearTokens, currentTokens, loadTokens, saveTokens } from "../session/tokenStore";
import type { AuthSession } from "./types";

function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return explicit.replace(/\/+$/, "");
  if (__DEV__) {
    // Emulator defaults. Physical phones need EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveBaseUrl();

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

export const NETWORK_ERROR_CODE = "network";

type SessionListener = (session: AuthSession | null) => void;
const sessionListeners = new Set<SessionListener>();

/** Notified when the client refreshes or loses the session on its own. */
export function onSessionChange(listener: SessionListener): () => void {
  sessionListeners.add(listener);
  return () => sessionListeners.delete(listener);
}

function emitSession(session: AuthSession | null) {
  sessionListeners.forEach((listener) => listener(session));
}

export function absoluteUrl(path: string): string {
  if (/^https?:\/\//.test(path)) return path;
  return `${API_BASE_URL}${path}`;
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

/** Single-flight refresh: concurrent 401s share one refresh round trip. */
async function refreshSession(): Promise<boolean> {
  if (!refreshing) {
    refreshing = (async () => {
      const tokens = currentTokens() ?? (await loadTokens());
      if (!tokens?.refreshToken) return false;
      try {
        const session = await rawRequest<AuthSession>("/auth/refresh", {
          method: "POST",
          body: { refreshToken: tokens.refreshToken },
          auth: false
        });
        await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
        emitSession(session);
        return true;
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) {
          await clearTokens();
          emitSession(null);
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
    throw new ApiError(0, "config", "L'URL de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const headers: Record<string, string> = { Accept: "application/json" };
  if (options.auth !== false) {
    const tokens = currentTokens() ?? (await loadTokens());
    if (tokens) headers.Authorization = `Bearer ${tokens.accessToken}`;
  }
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch {
    throw new ApiError(0, NETWORK_ERROR_CODE, "Connexion impossible. Vérifiez votre réseau et réessayez.");
  } finally {
    clearTimeout(timeout);
  }

  const text = await response.text();
  let payload: unknown = undefined;
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = undefined;
    }
  }
  if (!response.ok) {
    const data = (payload ?? {}) as { error?: string; code?: string };
    throw new ApiError(response.status, data.code ?? "http_error", data.error ?? `Erreur ${response.status}`);
  }
  return payload as T;
}

/**
 * Typed REST call. On a 401 the session is refreshed once and the request
 * retried; UI code never touches tokens or fetch directly.
 */
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

/** Ensures a fresh access token for the realtime connection. */
export async function freshAccessToken(): Promise<string | null> {
  const tokens = currentTokens() ?? (await loadTokens());
  return tokens?.accessToken ?? null;
}

export { refreshSession };
