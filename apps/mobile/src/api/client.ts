import { API_BASE_URL } from "../config";
import { ApiError } from "../lib/errors";
import { clearTokens, getTokens, saveTokens } from "./tokenStore";
import type { AuthResponse } from "./types";

type RequestOptions = {
  body?: unknown;
  form?: FormData;
  query?: Record<string, string | number | undefined>;
  /** false pour les endpoints publics (login, register...). */
  auth?: boolean;
};

let onSessionExpired: (() => void) | null = null;
let onTokenRefreshed: ((accessToken: string) => void) | null = null;
let refreshInFlight: Promise<string | null> | null = null;

/** Declare le callback appele quand la session ne peut plus etre renouvelee. */
export function setSessionExpiredHandler(handler: (() => void) | null): void {
  onSessionExpired = handler;
}

/** Declare le callback appele apres un renouvellement reussi de l'access token. */
export function setTokenRefreshedHandler(handler: ((accessToken: string) => void) | null): void {
  onTokenRefreshed = handler;
}

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  const url = new URL(API_BASE_URL + path);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined && value !== "") {
        url.searchParams.set(key, String(value));
      }
    }
  }
  return url.toString();
}

/** Renouvelle l'access token (une seule requete a la fois). Renvoie le nouveau token ou null. */
async function refreshAccessToken(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        return null;
      }
      try {
        const response = await fetch(buildUrl("/auth/refresh"), {
          method: "POST",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ refreshToken: tokens.refreshToken }),
        });
        if (response.status === 401) {
          await clearTokens();
          onSessionExpired?.();
          return null;
        }
        if (!response.ok) {
          return null; // erreur transitoire: on garde la session
        }
        const data = (await response.json()) as AuthResponse;
        await saveTokens({ accessToken: data.accessToken, refreshToken: data.refreshToken });
        onTokenRefreshed?.(data.accessToken);
        return data.accessToken;
      } catch {
        return null;
      }
    })().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

async function send(method: string, path: string, options: RequestOptions, token: string | null): Promise<Response> {
  const headers: Record<string, string> = { Accept: "application/json" };
  let body: string | FormData | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  try {
    return await fetch(buildUrl(path, options.query), { method, headers, body });
  } catch {
    throw new ApiError(0, "Connexion impossible. Vérifiez votre réseau et réessayez.");
  }
}

export async function request<T>(method: string, path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let token: string | null = null;
  if (useAuth) {
    token = (await getTokens())?.accessToken ?? null;
  }

  let response = await send(method, path, options, token);
  if (response.status === 401 && useAuth) {
    const fresh = await refreshAccessToken();
    if (fresh) {
      response = await send(method, path, options, fresh);
    }
  }

  if (!response.ok) {
    let message = `HTTP ${response.status}`;
    try {
      const data = (await response.json()) as { error?: string };
      if (data.error) {
        message = data.error;
      }
    } catch {
      // corps non JSON
    }
    throw new ApiError(response.status, message);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

/** Source d'image authentifiee pour expo-image (les photos ne sont jamais publiques). */
export function photoSource(
  photoId: string,
  accessToken: string | null,
): { uri: string; headers: Record<string, string> } {
  return {
    uri: `${API_BASE_URL}/photos/${photoId}`,
    headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {},
  };
}
