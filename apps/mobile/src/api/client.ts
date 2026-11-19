import { API_BASE_URL } from "./config";
import { clearTokens, loadTokens, saveTokens, type Tokens } from "./tokenStore";

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

export const NETWORK_ERROR_MESSAGE = "Connexion impossible. Vérifiez votre réseau puis réessayez.";

type SessionListener = () => void;
let onSessionExpired: SessionListener | null = null;

/** Registered by the session provider to react when refresh fails. */
export function setSessionExpiredListener(listener: SessionListener | null): void {
  onSessionExpired = listener;
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

let refreshing: Promise<Tokens | null> | null = null;

/** Single-flight refresh: concurrent 401s share one refresh request. */
async function refreshTokens(): Promise<Tokens | null> {
  if (!refreshing) {
    refreshing = (async () => {
      const current = await loadTokens();
      if (!current) return null;
      try {
        const res = await rawFetch("/auth/refresh", { method: "POST", body: { refreshToken: current.refreshToken } }, null);
        if (!res.ok) {
          await clearTokens();
          return null;
        }
        const data = (await res.json()) as Tokens;
        const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
        await saveTokens(next);
        return next;
      } catch {
        // Network failure: keep tokens, the next call will retry.
        return current;
      } finally {
        refreshing = null;
      }
    })();
  }
  return refreshing;
}

async function rawFetch(path: string, options: RequestOptions, accessToken: string | null): Promise<Response> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "config", "L'URL de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
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

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  const abort = () => controller.abort();
  options.signal?.addEventListener("abort", abort, { once: true });
  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (err) {
    if (options.signal?.aborted) throw err;
    throw new ApiError(0, "network", NETWORK_ERROR_MESSAGE);
  } finally {
    clearTimeout(timeout);
    options.signal?.removeEventListener("abort", abort);
  }
}

async function toApiError(res: Response): Promise<ApiError> {
  let code = "http_error";
  let message = `Erreur ${res.status}`;
  try {
    const data = (await res.json()) as { error?: string; code?: string };
    if (data.error) message = data.error;
    if (data.code) code = data.code;
  } catch {
    // Non JSON error body.
  }
  return new ApiError(res.status, code, message);
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let tokens = useAuth ? await loadTokens() : null;
  let res = await rawFetch(path, options, tokens?.accessToken ?? null);

  if (res.status === 401 && useAuth && tokens) {
    tokens = await refreshTokens();
    if (!tokens) {
      onSessionExpired?.();
      throw new ApiError(401, "unauthorized", "Votre session a expiré, reconnectez-vous.");
    }
    res = await rawFetch(path, options, tokens.accessToken);
    if (res.status === 401) {
      await clearTokens();
      onSessionExpired?.();
    }
  }

  if (!res.ok) {
    throw await toApiError(res);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof ApiError) return translate(error);
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

// User-facing messages for the codes returned by the API.
const messages: Record<string, string> = {
  invalid_credentials: "E-mail ou mot de passe incorrect.",
  email_taken: "Un compte existe déjà avec cet e-mail.",
  weak_password: "Le mot de passe doit contenir au moins 8 caractères, dont une lettre et un chiffre.",
  invalid_code: "Ce code est invalide ou a expiré.",
  rate_limited: "Trop de tentatives. Patientez un instant avant de réessayer.",
  underage: "Vous devez avoir au moins 18 ans pour utiliser Lueur.",
  birthdate_locked: "La date de naissance ne peut pas être modifiée.",
  profile_incomplete: "Ajoutez au moins une photo pour commencer à découvrir des profils.",
  photo_limit: "Vous pouvez ajouter jusqu'à 6 photos.",
  last_photo: "Gardez au moins une photo tant que votre profil est visible.",
  unsupported_type: "Seules les photos JPEG et PNG sont acceptées.",
  file_too_large: "Cette photo est trop lourde (8 Mo maximum).",
  conversation_closed: "Cette conversation est terminée.",
  already_swiped: "Vous avez déjà répondu à ce profil.",
  invalid_password: "Mot de passe incorrect.",
  internal_error: "Le serveur a rencontré un problème. Réessayez dans un instant."
};

function translate(error: ApiError): string {
  return messages[error.code] ?? error.message;
}
