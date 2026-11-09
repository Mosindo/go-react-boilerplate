import { API_URL } from "../config";
import { isExpiring } from "../lib/jwt";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import type { AuthResponse } from "./types";

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

const TIMEOUT_MS = 15_000;

let onSessionExpired: (() => void) | null = null;

/** The auth layer registers a callback fired when the refresh token is rejected. */
export function setSessionExpiredHandler(handler: (() => void) | null): void {
  onSessionExpired = handler;
}

let refreshInFlight: Promise<string | null> | null = null;

async function refreshTokens(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = await getTokens();
      if (!tokens?.refreshToken) {
        return null;
      }
      try {
        const res = await fetch(`${API_URL}/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refreshToken: tokens.refreshToken })
        });
        if (!res.ok) {
          if (res.status === 401) {
            await clearTokens();
            onSessionExpired?.();
          }
          return null;
        }
        const data = (await res.json()) as AuthResponse;
        await saveTokens({ accessToken: data.accessToken, refreshToken: data.refreshToken });
        return data.accessToken;
      } catch {
        return null; // offline: keep the session, retry later
      } finally {
        refreshInFlight = null;
      }
    })();
  }
  return refreshInFlight;
}

/** Returns an access token that is valid for at least a few more seconds, refreshing if needed. */
export async function getValidAccessToken(): Promise<string | null> {
  const tokens = await getTokens();
  if (!tokens) {
    return null;
  }
  if (isExpiring(tokens.accessToken, Date.now())) {
    return (await refreshTokens()) ?? tokens.accessToken;
  }
  return tokens.accessToken;
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
};

async function send(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
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
    const aborted = (error as { name?: string }).name === "AbortError";
    throw new ApiError(0, aborted ? "Le serveur met trop de temps à répondre." : "Connexion impossible. Vérifiez votre réseau.");
  } finally {
    clearTimeout(timer);
  }
}

async function parseError(res: Response): Promise<ApiError> {
  let message = `Erreur ${res.status}`;
  try {
    const data = (await res.json()) as { error?: unknown };
    if (typeof data.error === "string") {
      message = data.error;
    }
  } catch {
    // Non-JSON error body: keep the generic message.
  }
  return new ApiError(res.status, message);
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  if (!API_URL) {
    throw new ApiError(0, "EXPO_PUBLIC_API_URL n'est pas configurée pour ce build.");
  }
  const useAuth = options.auth !== false;
  let token = useAuth ? await getValidAccessToken() : null;
  let res = await send(path, options, token);
  if (res.status === 401 && useAuth) {
    token = await refreshTokens();
    if (token) {
      res = await send(path, options, token);
    }
  }
  if (!res.ok) {
    throw await parseError(res);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
