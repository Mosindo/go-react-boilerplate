import { Platform } from "react-native";
import { loadTokens, persistTokens, type AuthTokens } from "./tokenStore";
import type { AuthResponse } from "./types";

function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return explicit.replace(/\/+$/, "");
  if (__DEV__) {
    // Android emulators reach the host through 10.0.2.2. Physical devices
    // must set EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveBaseUrl();

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
  constructor(message = "Connexion impossible. Vérifiez votre réseau.") {
    super(message);
    this.name = "NetworkError";
  }
}

// ---------------------------------------------------------------------------
// Session state (single source of truth for tokens)
// ---------------------------------------------------------------------------

let tokens: AuthTokens | null = null;
const sessionListeners = new Set<(t: AuthTokens | null) => void>();

export function onSessionChange(listener: (t: AuthTokens | null) => void): () => void {
  sessionListeners.add(listener);
  return () => sessionListeners.delete(listener);
}

export function getTokens(): AuthTokens | null {
  return tokens;
}

export async function setSession(next: AuthTokens | null): Promise<void> {
  tokens = next;
  sessionListeners.forEach((l) => l(next));
  await persistTokens(next);
}

export async function restoreSession(): Promise<AuthTokens | null> {
  tokens = await loadTokens();
  return tokens;
}

let refreshing: Promise<boolean> | null = null;

/** Rotates the refresh token once even if many requests fail concurrently. */
export function refreshSession(): Promise<boolean> {
  if (refreshing) return refreshing;
  const current = tokens;
  if (!current?.refreshToken) return Promise.resolve(false);
  refreshing = (async () => {
    try {
      const res = await rawFetch("/auth/refresh", { method: "POST", body: JSON.stringify({ refreshToken: current.refreshToken }) }, null);
      if (!res.ok) {
        if (res.status === 401) await setSession(null);
        return false;
      }
      const data = (await res.json()) as AuthResponse;
      await setSession({ accessToken: data.accessToken, refreshToken: data.refreshToken });
      return true;
    } catch {
      return false;
    } finally {
      refreshing = null;
    }
  })();
  return refreshing;
}

// ---------------------------------------------------------------------------
// Requests
// ---------------------------------------------------------------------------

const TIMEOUT_MS = 15_000;

async function rawFetch(path: string, init: RequestInit, accessToken: string | null, timeoutMs = TIMEOUT_MS): Promise<Response> {
  if (!API_BASE_URL) throw new NetworkError("L'URL de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (init.body && !(init.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(`${API_BASE_URL}${path}`, { ...init, headers, signal: controller.signal });
  } catch (err) {
    if ((err as { name?: string }).name === "AbortError") throw new NetworkError("Le serveur met trop de temps à répondre.");
    throw new NetworkError();
  } finally {
    clearTimeout(timer);
  }
}

async function toApiError(res: Response): Promise<ApiError> {
  let code = "http_error";
  let message = `Erreur ${res.status}`;
  try {
    const body = (await res.json()) as { error?: string; code?: string };
    code = body.code ?? code;
    message = body.error ?? message;
  } catch {
    /* non JSON error */
  }
  return new ApiError(res.status, code, message);
}

type RequestOptions = { method?: string; body?: unknown; form?: FormData; auth?: boolean; timeoutMs?: number };

export async function api<T = void>(path: string, opts: RequestOptions = {}): Promise<T> {
  const init: RequestInit = {
    method: opts.method ?? (opts.body || opts.form ? "POST" : "GET"),
    body: opts.form ?? (opts.body !== undefined ? JSON.stringify(opts.body) : undefined)
  };
  const useAuth = opts.auth !== false;
  let res = await rawFetch(path, init, useAuth ? (tokens?.accessToken ?? null) : null, opts.timeoutMs);

  if (res.status === 401 && useAuth && tokens?.refreshToken) {
    if (await refreshSession()) {
      res = await rawFetch(path, init, tokens?.accessToken ?? null, opts.timeoutMs);
    }
  }
  if (!res.ok) {
    const error = await toApiError(res);
    if (res.status === 401 && useAuth) await setSession(null);
    throw error;
  }
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

/** Photo URLs are signed relative paths; the cache key stays the photo id. */
export function mediaSource(photo: { id: string; url: string }) {
  return { uri: `${API_BASE_URL}${photo.url}`, cacheKey: `photo-${photo.id}-${photo.url.split("?")[0]}` };
}

export function realtimeUrl(ticket: string): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/realtime?ticket=${encodeURIComponent(ticket)}`;
}

/** Human readable message for any thrown error. */
export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof ApiError || error instanceof NetworkError) return translate(error);
  return fallback;
}

const messages: Record<string, string> = {
  invalid_credentials: "Email ou mot de passe incorrect.",
  email_exists: "Un compte existe déjà avec cet email.",
  weak_password: "Le mot de passe doit contenir 8 caractères minimum, dont une lettre et un chiffre.",
  invalid_email: "Adresse email invalide.",
  invalid_reset_code: "Code invalide ou expiré.",
  invalid_password: "Mot de passe incorrect.",
  rate_limited: "Trop de tentatives. Patientez un instant.",
  underage: "Lueur est réservée aux personnes majeures (18 ans et plus).",
  invalid_image: "Ce fichier n'est pas une image valide (JPEG, PNG ou WebP, 200 px minimum).",
  file_too_large: "Photo trop lourde (10 Mo maximum).",
  photo_limit: "Vous avez déjà 6 photos.",
  profile_incomplete: "Complétez votre profil pour découvrir des personnes.",
  profile_unavailable: "Ce profil n'est plus disponible.",
  already_swiped: "Vous avez déjà répondu à ce profil.",
  not_found: "Élément introuvable ou supprimé.",
  suspended: "Votre compte est suspendu.",
  internal_error: "Le serveur a rencontré un problème. Réessayez."
};

function translate(error: ApiError | NetworkError): string {
  if (error instanceof NetworkError) return error.message;
  if (error.code === "validation_failed") return validationMessage(error.message);
  return messages[error.code] ?? error.message;
}

function validationMessage(raw: string): string {
  if (raw.includes("birthdate cannot be changed")) return "La date de naissance ne peut pas être modifiée.";
  if (raw.includes("first name")) return "Le prénom doit contenir uniquement des lettres (30 max).";
  if (raw.includes("too long")) return "Un des champs est trop long.";
  if (raw.includes("interests")) return "8 centres d'intérêt maximum.";
  if (raw.includes("height")) return "La taille doit être comprise entre 120 et 230 cm.";
  return "Certaines informations sont invalides.";
}
