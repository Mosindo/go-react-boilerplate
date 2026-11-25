import { API_BASE_URL } from "./config";
import { clearTokens, currentTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

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

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

type SessionListener = () => void;
const sessionExpiredListeners = new Set<SessionListener>();

/** Subscribe to "session expired" events (refresh failed). */
export function onSessionExpired(listener: SessionListener): () => void {
  sessionExpiredListeners.add(listener);
  return () => {
    sessionExpiredListeners.delete(listener);
  };
}

let refreshInFlight: Promise<AuthTokens | null> | null = null;

async function refreshTokens(): Promise<AuthTokens | null> {
  const tokens = currentTokens();
  if (!tokens?.refreshToken) {
    return null;
  }
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      try {
        const res = await fetch(`${API_BASE_URL}/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ refreshToken: tokens.refreshToken })
        });
        if (!res.ok) {
          return null;
        }
        const payload = (await res.json()) as { accessToken: string; refreshToken: string };
        const next = { accessToken: payload.accessToken, refreshToken: payload.refreshToken };
        await saveTokens(next);
        return next;
      } catch {
        return null;
      } finally {
        refreshInFlight = null;
      }
    })();
  }
  return refreshInFlight;
}

async function expireSession() {
  await clearTokens();
  sessionExpiredListeners.forEach((listener) => listener());
}

async function parseError(res: Response): Promise<ApiError> {
  let code = "http_error";
  let message = `Erreur ${res.status}`;
  try {
    const payload = (await res.json()) as { error?: string; code?: string };
    code = payload.code ?? code;
    message = payload.error ?? message;
  } catch {
    // Non-JSON error body.
  }
  return new ApiError(res.status, code, message);
}

async function send(path: string, options: RequestOptions, accessToken: string | null): Promise<Response> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "not_configured", "EXPO_PUBLIC_API_URL n'est pas configurée pour ce build.");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`;
  }

  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (error) {
    if ((error as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "timeout", "Le serveur met trop de temps à répondre. Réessayez.");
    }
    throw new ApiError(0, "network", "Connexion impossible. Vérifiez votre réseau puis réessayez.");
  } finally {
    clearTimeout(timeout);
  }
}

/**
 * Single entry point for every API call. Authenticated requests transparently
 * refresh an expired access token once, and notify listeners when the
 * session cannot be renewed.
 */
export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let res = await send(path, options, useAuth ? (currentTokens()?.accessToken ?? null) : null);

  if (useAuth && res.status === 401 && currentTokens()) {
    const next = await refreshTokens();
    if (!next) {
      await expireSession();
      throw await parseError(res);
    }
    res = await send(path, options, next.accessToken);
    if (res.status === 401) {
      await expireSession();
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

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof ApiError) {
    return translateError(error) ?? error.message;
  }
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}

const messagesByCode: Record<string, string> = {
  email_exists: "Un compte existe déjà avec cet e-mail.",
  invalid_email: "Adresse e-mail invalide.",
  weak_password: "Le mot de passe doit contenir 8 à 72 caractères, dont une lettre et un chiffre.",
  invalid_credentials: "E-mail ou mot de passe incorrect.",
  invalid_reset_code: "Ce code est invalide ou expiré.",
  wrong_password: "Mot de passe incorrect.",
  rate_limited: "Trop de tentatives. Patientez un peu avant de réessayer.",
  underage: "Vous devez avoir au moins 18 ans pour utiliser Lueur.",
  invalid_birthdate: "Date de naissance invalide.",
  invalid_first_name: "Le prénom doit contenir uniquement des lettres (40 max).",
  invalid_bio: "La bio est limitée à 500 caractères.",
  invalid_interests: "Choisissez jusqu'à 10 centres d'intérêt.",
  invalid_preferences: "Préférences invalides.",
  profile_required: "Créez d'abord votre profil.",
  profile_incomplete: "Complétez votre profil pour découvrir des personnes.",
  photo_limit: "Vous pouvez ajouter jusqu'à 6 photos.",
  unsupported_image: "Format d'image non pris en charge (JPEG ou PNG).",
  image_too_small: "Image trop petite (200×200 minimum).",
  image_too_large: "Image trop volumineuse.",
  profile_unavailable: "Ce profil n'est plus disponible.",
  conversation_not_found: "Cette conversation n'est plus disponible.",
  message_too_long: "Message trop long (2000 caractères max).",
  network: "Connexion impossible. Vérifiez votre réseau puis réessayez.",
  timeout: "Le serveur met trop de temps à répondre. Réessayez.",
  internal_error: "Erreur du serveur. Réessayez dans un instant."
};

function translateError(error: ApiError): string | undefined {
  return messagesByCode[error.code];
}
