import { Platform } from "react-native";
import { clearTokens, getAccessToken, getRefreshToken, saveTokens } from "../store/tokenStore";
import { emitSessionExpired } from "./sessionEvents";

const REQUEST_TIMEOUT_MS = 15_000;

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function developmentFallbackBaseUrl(): string {
  return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return normalizeBaseUrl(explicit);
  return __DEV__ ? developmentFallbackBaseUrl() : "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export class ApiError extends Error {
  public status: number;
  public code: string;

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

/** Human-readable message for any thrown value, in French for end users. */
export function describeError(error: unknown, fallback = "Une erreur est survenue. Réessayez."): string {
  if (error instanceof ApiError) return error.message;
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

const FRENCH_MESSAGES: Record<string, string> = {
  invalid_credentials: "E-mail ou mot de passe incorrect.",
  email_exists: "Un compte existe déjà avec cet e-mail.",
  invalid_email: "Adresse e-mail invalide.",
  invalid_password: "Le mot de passe doit contenir entre 8 et 72 caractères.",
  weak_password: "Le mot de passe doit mélanger des lettres et des chiffres ou symboles.",
  rate_limited: "Trop de tentatives. Patientez un instant avant de réessayer.",
  invalid_reset_code: "Code invalide ou expiré.",
  wrong_password: "Mot de passe incorrect.",
  underage: "Vous devez avoir au moins 18 ans pour utiliser l'application.",
  invalid_first_name: "Le prénom doit contenir 1 à 40 lettres.",
  invalid_birthdate: "Date de naissance invalide.",
  profile_incomplete: "Complétez votre profil et ajoutez une photo d'abord.",
  too_many_photos: "Vous pouvez ajouter jusqu'à 6 photos.",
  last_photo: "Gardez au moins une photo sur votre profil.",
  unsupported_media_type: "Seules les images JPEG et PNG sont acceptées.",
  image_too_small: "La photo doit faire au moins 200 × 200 pixels.",
  image_too_large: "Cette photo est trop volumineuse (8 Mo maximum).",
  invalid_image: "Ce fichier n'est pas une image valide.",
  account_suspended: "Ce compte a été suspendu.",
  not_found: "Introuvable. Cet élément n'est plus disponible."
};

async function parseError(response: Response): Promise<ApiError> {
  let code = "http_error";
  let message = `Erreur ${response.status}`;
  try {
    const data = (await response.json()) as { error?: string; code?: string };
    if (data.code) code = data.code;
    message = FRENCH_MESSAGES[code] ?? data.error ?? message;
  } catch {
    // Non-JSON error body (proxy error page, ...): keep the generic message.
  }
  if (response.status >= 500) message = "Le serveur rencontre un problème. Réessayez dans un instant.";
  return new ApiError(response.status, code, message);
}

let refreshInFlight: Promise<boolean> | null = null;

/** Exchanges the refresh token for a new pair; concurrent callers share one request. */
function refreshSession(): Promise<boolean> {
  if (refreshInFlight) return refreshInFlight;
  refreshInFlight = (async () => {
    const refreshToken = getRefreshToken();
    if (!refreshToken) return false;
    try {
      const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ refreshToken })
      });
      if (!response.ok) return false;
      const data = (await response.json()) as { accessToken: string; refreshToken: string };
      await saveTokens({ accessToken: data.accessToken, refreshToken: data.refreshToken });
      return true;
    } catch {
      return false;
    }
  })().finally(() => {
    refreshInFlight = null;
  });
  return refreshInFlight;
}

type RequestOptions = {
  method?: string;
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
};

async function send(path: string, options: RequestOptions): Promise<Response> {
  if (!API_BASE_URL) throw new Error("L'adresse du serveur n'est pas configurée (EXPO_PUBLIC_API_URL).");
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  const token = getAccessToken();
  if (options.auth !== false && token) headers.Authorization = `Bearer ${token}`;
  if (options.body !== undefined) headers["Content-Type"] = "application/json";

  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body: options.form ?? (options.body !== undefined ? JSON.stringify(options.body) : undefined),
      signal: controller.signal
    });
  } catch (error) {
    if ((error as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "timeout", "Le serveur met trop de temps à répondre.");
    }
    throw new ApiError(0, "network", "Connexion impossible. Vérifiez votre réseau.");
  } finally {
    clearTimeout(timeout);
  }
}

export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  let response = await send(path, options);

  if (response.status === 401 && options.auth !== false && getRefreshToken()) {
    if (await refreshSession()) {
      response = await send(path, options);
    } else {
      await clearTokens();
      emitSessionExpired();
    }
  }

  if (!response.ok) throw await parseError(response);
  if (response.status === 204) return undefined as T;
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export function wsUrl(ticket: string): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws?ticket=${encodeURIComponent(ticket)}`;
}

/** Photo URLs are relative to the API and protected: they require the bearer token. */
export function absoluteUrl(path: string): string {
  return `${API_BASE_URL}${path}`;
}

export function authHeaders(): Record<string, string> {
  const token = getAccessToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

/** Authenticated binary download (photos on web, where <img> cannot send an Authorization header). */
export async function apiBlob(path: string): Promise<Blob> {
  let response = await send(path, {});
  if (response.status === 401 && getRefreshToken() && (await refreshSession())) {
    response = await send(path, {});
  }
  if (!response.ok) throw await parseError(response);
  return response.blob();
}
