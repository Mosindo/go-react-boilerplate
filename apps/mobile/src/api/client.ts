import { Platform } from "react-native";
import { clearTokens, getCachedTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return normalizeBaseUrl(explicit);
  // Development fallbacks only. Physical phones must set EXPO_PUBLIC_API_URL
  // to the LAN IP of the machine running the API (see README).
  return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
}

export const API_BASE_URL = resolveApiBaseUrl();

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

type SessionListener = {
  onTokens: (tokens: AuthTokens) => void;
  onExpired: () => void;
};

let listener: SessionListener | null = null;

export function setSessionListener(next: SessionListener | null): void {
  listener = next;
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

type ErrorBody = { error?: string; code?: string };

async function send(path: string, options: RequestOptions, accessToken: string | null): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });
  const headers: Record<string, string> = { Accept: "application/json" };
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`;
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
    if (controller.signal.aborted && !options.signal?.aborted) {
      throw new ApiError(0, "timeout", "Le serveur met trop de temps à répondre. Réessayez.");
    }
    if (options.signal?.aborted) throw error;
    throw new ApiError(0, "network", "Connexion impossible. Vérifiez votre réseau et réessayez.");
  } finally {
    clearTimeout(timeout);
  }
}

let refreshing: Promise<AuthTokens | null> | null = null;

/** Refreshes the session once even if several requests fail at the same time. */
function refreshTokens(): Promise<AuthTokens | null> {
  if (refreshing) return refreshing;
  refreshing = (async () => {
    const current = getCachedTokens();
    if (!current?.refreshToken) return null;
    try {
      const res = await send("/auth/refresh", { method: "POST", body: { refreshToken: current.refreshToken } }, null);
      if (!res.ok) {
        if (res.status === 401) {
          await clearTokens();
          listener?.onExpired();
        }
        return null;
      }
      const data = (await res.json()) as { accessToken: string; refreshToken: string };
      const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
      await saveTokens(next);
      listener?.onTokens(next);
      return next;
    } catch {
      return null;
    } finally {
      refreshing = null;
    }
  })();
  return refreshing;
}

export async function refreshSessionNow(): Promise<boolean> {
  return (await refreshTokens()) !== null;
}

async function toError(res: Response): Promise<ApiError> {
  let body: ErrorBody = {};
  try {
    body = (await res.json()) as ErrorBody;
  } catch {
    // non JSON error
  }
  const fallback = res.status >= 500 ? "Le service rencontre un problème. Réessayez plus tard." : "Requête refusée.";
  const message = res.status === 429 ? "Trop de requêtes. Patientez un instant." : (body.error ?? fallback);
  return new ApiError(res.status, body.code ?? "error", message);
}

export async function api<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let res = await send(path, options, useAuth ? (getCachedTokens()?.accessToken ?? null) : null);
  if (res.status === 401 && useAuth) {
    const next = await refreshTokens();
    if (next) res = await send(path, options, next.accessToken);
  }
  if (!res.ok) throw await toError(res);
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export function apiErrorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof ApiError) return error.message;
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}
