import { clearTokens, getTokens, saveTokens, type AuthTokens } from "../store/tokenStore";
import { API_BASE_URL } from "./config";

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

type SessionHandlers = {
  onTokens: (tokens: AuthTokens) => void;
  onExpired: () => void;
};

let sessionHandlers: SessionHandlers | null = null;

/** Lets the auth provider react to silent refreshes and expirations. */
export function configureSession(handlers: SessionHandlers | null): void {
  sessionHandlers = handlers;
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

type ErrorPayload = { error?: string; code?: string };

let refreshInFlight: Promise<AuthTokens | null> | null = null;

async function refreshTokens(): Promise<AuthTokens | null> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const current = await getTokens();
      if (!current?.refreshToken) {
        return null;
      }
      try {
        const res = await rawRequest<{ accessToken: string; refreshToken: string }>("/auth/refresh", {
          method: "POST",
          body: { refreshToken: current.refreshToken },
          auth: false
        });
        const next = { accessToken: res.accessToken, refreshToken: res.refreshToken };
        await saveTokens(next);
        sessionHandlers?.onTokens(next);
        return next;
      } catch (err) {
        if (err instanceof ApiError && (err.status === 401 || err.status === 400)) {
          await clearTokens();
          sessionHandlers?.onExpired();
        }
        return null;
      }
    })().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

async function rawRequest<T>(path: string, options: RequestOptions, token?: string): Promise<T> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "not_configured", "L'adresse du serveur n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  if (options.signal) {
    if (options.signal.aborted) {
      controller.abort();
    } else {
      options.signal.addEventListener("abort", () => controller.abort(), { once: true });
    }
  }

  const headers: Record<string, string> = { Accept: "application/json" };
  let body: BodyInit | undefined;
  if (options.body instanceof FormData) {
    body = options.body;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (err) {
    if ((err as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "timeout", "Le serveur met trop de temps à répondre. Réessayez.");
    }
    throw new ApiError(0, "network", "Connexion impossible. Vérifiez votre réseau et réessayez.");
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
    const err = (payload ?? {}) as ErrorPayload;
    throw new ApiError(response.status, err.code ?? `http_${response.status}`, err.error ?? `Erreur serveur (${response.status})`);
  }
  return payload as T;
}

/**
 * Performs an API request. Authenticated requests transparently refresh an
 * expired access token once (single-flight) and retry.
 */
export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  const auth = options.auth ?? true;
  if (!auth) {
    return rawRequest<T>(path, options);
  }
  const tokens = await getTokens();
  try {
    return await rawRequest<T>(path, options, tokens?.accessToken);
  } catch (err) {
    if (!(err instanceof ApiError) || err.status !== 401 || !tokens) {
      throw err;
    }
    const refreshed = await refreshTokens();
    if (!refreshed) {
      throw new ApiError(401, "session_expired", "Votre session a expiré. Reconnectez-vous.");
    }
    return rawRequest<T>(path, options, refreshed.accessToken);
  }
}

/** Returns a valid access token (refreshing if needed) for the realtime socket. */
export async function freshAccessToken(): Promise<string | null> {
  const refreshed = await refreshTokens();
  return refreshed?.accessToken ?? null;
}

/** Human readable message for any error thrown by the API layer. */
export function errorMessage(err: unknown, fallback = "Une erreur est survenue. Réessayez."): string {
  if (err instanceof ApiError) {
    return translateError(err) ?? err.message;
  }
  if (err instanceof Error && err.message) {
    return err.message;
  }
  return fallback;
}

const translations: Record<string, string> = {
  email_exists: "Un compte existe déjà avec cet e-mail.",
  weak_password: "Le mot de passe doit contenir au moins 8 caractères, dont une lettre et un chiffre.",
  invalid_code: "Code invalide ou expiré.",
  wrong_password: "Mot de passe incorrect.",
  underage: "Vous devez avoir au moins 18 ans pour utiliser Lueur.",
  rate_limited: "Trop de tentatives. Patientez quelques minutes.",
  photo_limit: "Vous pouvez ajouter jusqu'à 6 photos.",
  invalid_image: "Ce fichier n'est pas une image valide (JPEG, PNG ou WebP, 200×200 min).",
  file_too_large: "Image trop lourde (10 Mo maximum).",
  profile_unavailable: "Ce profil n'est plus disponible.",
  conversation_not_found: "Cette conversation n'est plus disponible.",
  profile_incomplete: "Complétez votre profil (avec au moins une photo) pour continuer."
};

function translateError(err: ApiError): string | null {
  if (translations[err.code]) {
    return translations[err.code];
  }
  if (err.status === 401 && err.code === "invalid_token") {
    return "Votre session a expiré. Reconnectez-vous.";
  }
  if (err.code === "http_401" || err.message === "invalid credentials") {
    return "E-mail ou mot de passe incorrect.";
  }
  if (err.status >= 500) {
    return "Le service rencontre un problème. Réessayez dans un instant.";
  }
  return null;
}
