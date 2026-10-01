import { Platform } from "react-native";
import { clearTokens, loadTokens, saveTokens, type StoredTokens } from "../lib/tokenStore";

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }

  get isNetwork(): boolean {
    return this.status === 0;
  }
}

function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return explicit.replace(/\/+$/, "");
  if (Platform.OS === "android") return "http://10.0.2.2:18080";
  return "http://localhost:18080";
}

export const API_BASE_URL = resolveBaseUrl();

const REQUEST_TIMEOUT_MS = 15_000;

type SessionListeners = { onExpired: () => void };
let listeners: SessionListeners = { onExpired: () => undefined };

/** Registered once by the auth provider so a dead refresh token logs the user out. */
export function setSessionListeners(next: SessionListeners): void {
  listeners = next;
}

let refreshing: Promise<StoredTokens | null> | null = null;

/** Single-flight refresh: concurrent 401s share one rotation (refresh tokens are single-use). */
async function refreshTokens(): Promise<StoredTokens | null> {
  if (refreshing) return refreshing;
  refreshing = (async () => {
    const current = await loadTokens();
    if (!current) return null;
    try {
      const res = await rawFetch("/auth/refresh", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refreshToken: current.refreshToken })
      });
      if (!res.ok) {
        if (res.status === 401) {
          await clearTokens();
          listeners.onExpired();
        }
        return null;
      }
      const body = (await res.json()) as { accessToken: string; refreshToken: string };
      const next = { accessToken: body.accessToken, refreshToken: body.refreshToken };
      await saveTokens(next);
      return next;
    } catch {
      return null; // offline: keep the session, the next request will retry
    }
  })().finally(() => {
    refreshing = null;
  });
  return refreshing;
}

async function rawFetch(path: string, init: RequestInit): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    return await fetch(`${API_BASE_URL}${path}`, { ...init, signal: controller.signal });
  } catch {
    throw new ApiError(0, "network", "Connexion impossible. Vérifiez votre réseau et réessayez.");
  } finally {
    clearTimeout(timer);
  }
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
};

async function toError(res: Response): Promise<ApiError> {
  let code = "http_error";
  let message = `Erreur ${res.status}`;
  try {
    const data = (await res.json()) as { error?: string; code?: string };
    if (data.error) message = data.error;
    if (data.code) code = data.code;
  } catch {
    // non-JSON error body
  }
  if (res.status === 429) message = "Trop de requêtes, patientez un instant.";
  return new ApiError(res.status, code, message);
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, form, auth = true } = options;

  const send = async (tokens: StoredTokens | null): Promise<Response> => {
    const headers: Record<string, string> = { Accept: "application/json" };
    if (auth && tokens) headers.Authorization = `Bearer ${tokens.accessToken}`;
    let payload: BodyInit | undefined;
    if (form) {
      payload = form; // fetch sets the multipart boundary itself
    } else if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      payload = JSON.stringify(body);
    }
    return rawFetch(path, { method, headers, body: payload });
  };

  let tokens = auth ? await loadTokens() : null;
  let res = await send(tokens);
  if (res.status === 401 && auth && tokens) {
    tokens = await refreshTokens();
    if (tokens) res = await send(tokens);
  }
  if (!res.ok) throw await toError(res);
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

/** Fetches a protected image as a blob (used on web, where <img> cannot send auth headers). */
export async function fetchAuthedBlob(path: string): Promise<Blob> {
  let tokens = await loadTokens();
  const get = (t: StoredTokens | null) =>
    rawFetch(path, { headers: t ? { Authorization: `Bearer ${t.accessToken}` } : {} });
  let res = await get(tokens);
  if (res.status === 401 && tokens) {
    tokens = await refreshTokens();
    if (tokens) res = await get(tokens);
  }
  if (!res.ok) throw await toError(res);
  return res.blob();
}

export async function currentAccessToken(): Promise<string | null> {
  return (await loadTokens())?.accessToken ?? null;
}

/** Test helper: wait for any in-flight refresh. */
export function _refreshInFlight(): Promise<StoredTokens | null> | null {
  return refreshing;
}
