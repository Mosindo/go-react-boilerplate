import { Platform } from "react-native";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import { showGlobalError } from "../shared/feedback";

export type ApiErrorPayload = { error?: string; field?: string; code?: string };

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function developmentFallbackBaseUrl(): string {
  return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
}

function resolveApiBaseUrl(): string {
  const explicitBaseUrl = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicitBaseUrl) {
    return normalizeBaseUrl(explicitBaseUrl);
  }
  return __DEV__ ? developmentFallbackBaseUrl() : "";
}

export const API_BASE_URL = resolveApiBaseUrl();

const REQUEST_TIMEOUT_MS = 15_000;
const UPLOAD_TIMEOUT_MS = 60_000;

export class ApiError extends Error {
  public status: number;
  public field?: string;
  public code?: string;

  constructor(status: number, message: string, payload?: ApiErrorPayload | null) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.field = payload?.field;
    this.code = payload?.code;
  }
}

let sessionExpiredHandler: (() => void) | null = null;

/** Registered by the auth provider: called when the refresh token is no longer accepted. */
export function setSessionExpiredHandler(handler: (() => void) | null) {
  sessionExpiredHandler = handler;
}

let refreshInFlight: Promise<boolean> | null = null;

/** Exchanges the refresh token for a new pair. Concurrent callers share one request. */
function refreshSession(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = await getTokens();
      if (!tokens?.refreshToken) {
        return false;
      }
      try {
        const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ refreshToken: tokens.refreshToken })
        });
        if (!response.ok) {
          if (response.status === 401 || response.status === 403) {
            await clearTokens();
          }
          return false;
        }
        const body = (await response.json()) as { accessToken: string; refreshToken: string };
        await saveTokens({ accessToken: body.accessToken, refreshToken: body.refreshToken });
        return true;
      } catch {
        return false;
      }
    })().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

function networkHint(): string {
  if (!API_BASE_URL) {
    return "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).";
  }
  return "Connexion impossible. Vérifiez votre réseau puis réessayez.";
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
  /** Suppress the global error banner (the caller shows its own message). */
  quiet?: boolean;
};

async function send(path: string, options: RequestOptions, accessToken: string | null): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), options.form ? UPLOAD_TIMEOUT_MS : REQUEST_TIMEOUT_MS);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (accessToken && options.auth !== false) {
    headers.Authorization = `Bearer ${accessToken}`;
  }
  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body: options.form ?? (options.body !== undefined ? JSON.stringify(options.body) : undefined),
      signal: controller.signal
    });
  } finally {
    clearTimeout(timeoutId);
  }
}

export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  let response: Response;
  try {
    const tokens = options.auth === false ? null : await getTokens();
    response = await send(path, options, tokens?.accessToken ?? null);
    if (response.status === 401 && options.auth !== false && tokens) {
      if (await refreshSession()) {
        const fresh = await getTokens();
        response = await send(path, options, fresh?.accessToken ?? null);
      }
      if (response.status === 401) {
        await clearTokens();
        sessionExpiredHandler?.();
      }
    }
  } catch (err) {
    const aborted = (err as { name?: string }).name === "AbortError";
    const message = aborted ? "La requête a expiré. Réessayez." : networkHint();
    if (!options.quiet) {
      showGlobalError(message);
    }
    throw new ApiError(0, message);
  }

  if (!response.ok) {
    let payload: ApiErrorPayload | null = null;
    try {
      payload = (await response.json()) as ApiErrorPayload;
    } catch {
      payload = null;
    }
    const message = payload?.error ?? `Erreur ${response.status}`;
    if (response.status >= 500 && !options.quiet) {
      showGlobalError("Le service rencontre un problème. Réessayez dans un instant.");
    }
    if (response.status === 429 && !options.quiet) {
      showGlobalError("Trop de requêtes. Patientez quelques instants.");
    }
    throw new ApiError(response.status, message, payload);
  }

  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

/** Message safe to show to the user for any thrown value. */
export function errorMessage(err: unknown, fallback = "Une erreur est survenue."): string {
  if (err instanceof ApiError) {
    return translateApiMessage(err);
  }
  return err instanceof Error && err.message ? err.message : fallback;
}

const apiMessages: Record<string, string> = {
  "invalid credentials": "Email ou mot de passe incorrect.",
  "email already exists": "Un compte existe déjà avec cet email.",
  "account suspended": "Ce compte est suspendu.",
  "too many requests": "Trop de tentatives. Patientez quelques instants.",
  "invalid or expired reset token": "Ce code est invalide ou a expiré.",
  "photo limit reached (6 maximum)": "Vous avez déjà 6 photos.",
  "unsupported image type (JPEG, PNG or WebP)": "Format non pris en charge (JPEG, PNG ou WebP).",
  "image too large": "Cette image est trop volumineuse.",
  "image too small (200px minimum)": "Cette image est trop petite (200 px minimum).",
  "complete your profile first": "Terminez d'abord votre profil."
};

function translateApiMessage(err: ApiError): string {
  return apiMessages[err.message] ?? err.message;
}

/** Base URL for the realtime socket. */
export function websocketUrl(): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws`;
}
