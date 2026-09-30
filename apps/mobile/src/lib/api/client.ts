import { API_BASE_URL } from "./config";
import { session, type Tokens } from "./session";

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  timeoutMs?: number;
  signal?: AbortSignal;
};

// French user-facing messages for known API error codes.
const MESSAGES: Record<string, string> = {
  invalid_credentials: "Email ou mot de passe incorrect.",
  email_exists: "Un compte existe déjà avec cet email.",
  underage: "Vous devez avoir au moins 18 ans pour utiliser Lueur.",
  rate_limited: "Trop de tentatives. Patientez un instant avant de réessayer.",
  invalid_reset_code: "Code invalide ou expiré.",
  reset_unavailable: "La récupération de compte n'est pas disponible pour le moment.",
  profile_incomplete: "Complétez votre profil pour découvrir des personnes.",
  already_swiped: "Vous avez déjà répondu à ce profil.",
  photo_limit: "Vous pouvez ajouter jusqu'à 6 photos.",
  unsupported_media_type: "Format non pris en charge (JPEG, PNG ou WebP uniquement).",
  image_too_large: "Image trop lourde (10 Mo maximum).",
  invalid_password: "Mot de passe incorrect.",
  birthdate_locked: "La date de naissance ne peut plus être modifiée.",
  not_found: "Élément introuvable ou plus disponible.",
  internal_error: "Une erreur est survenue. Réessayez dans un instant."
};

let refreshing: Promise<Tokens | null> | null = null;
let sessionExpiredHandler: (() => void) | null = null;

export function onSessionExpired(handler: () => void): void {
  sessionExpiredHandler = handler;
}

/** Refreshes the session once even if many requests hit 401 together. */
export function refreshTokens(): Promise<Tokens | null> {
  if (!refreshing) {
    refreshing = (async () => {
      const tokens = session.get();
      if (!tokens?.refreshToken) {
        return null;
      }
      try {
        const res = await rawFetch("/auth/refresh", { method: "POST", body: { refreshToken: tokens.refreshToken } }, null);
        if (!res.ok) {
          return null;
        }
        const data = (await res.json()) as Tokens;
        const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
        await session.set(next);
        return next;
      } catch {
        // Network failure: keep the current session, let the caller retry.
        return tokens;
      }
    })().finally(() => {
      refreshing = null;
    });
  }
  return refreshing;
}

async function rawFetch(path: string, options: RequestOptions, accessToken: string | null): Promise<Response> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "not_configured", "L'adresse du serveur n'est pas configurée (EXPO_PUBLIC_API_URL).");
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
    return await fetch(`${API_BASE_URL}${path}`, { method: options.method ?? "GET", headers, body, signal: controller.signal });
  } catch (error) {
    if ((error as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "timeout", "Le serveur met trop de temps à répondre. Réessayez.");
    }
    throw new ApiError(0, "network", "Connexion impossible. Vérifiez votre réseau.");
  } finally {
    clearTimeout(timeout);
  }
}

async function toError(res: Response): Promise<ApiError> {
  let code = "http_" + res.status;
  let message = "";
  try {
    const payload = (await res.json()) as { error?: string; code?: string };
    code = payload.code ?? code;
    message = payload.error ?? "";
  } catch {
    // Non-JSON error body.
  }
  return new ApiError(res.status, code, MESSAGES[code] ?? (message || MESSAGES.internal_error));
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let res = await rawFetch(path, options, useAuth ? session.get()?.accessToken ?? null : null);

  if (res.status === 401 && useAuth && session.get()) {
    const refreshed = await refreshTokens();
    if (!refreshed) {
      await session.set(null);
      sessionExpiredHandler?.();
      throw new ApiError(401, "session_expired", "Votre session a expiré. Reconnectez-vous.");
    }
    res = await rawFetch(path, options, refreshed.accessToken);
  }

  if (!res.ok) {
    throw await toError(res);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export function errorMessage(error: unknown, fallback = MESSAGES.internal_error): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  return fallback;
}
