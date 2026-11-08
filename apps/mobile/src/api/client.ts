import { API_BASE_URL } from "../config";
import { clearTokens, currentTokens, saveTokens, type Tokens } from "../auth/tokenStore";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

type Query = Record<string, string | number | boolean | undefined | null>;

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  query?: Query;
  auth?: boolean;
  signal?: AbortSignal;
};

const TIMEOUT_MS = 15_000;

let sessionExpiredHandler: (() => void) | null = null;
export function onSessionExpired(handler: (() => void) | null): void {
  sessionExpiredHandler = handler;
}

// Messages shown to the user for the cases the server answers with a generic text.
const FRIENDLY: Record<number, string> = {
  401: "Ta session a expiré. Reconnecte-toi.",
  403: "Action non autorisée.",
  404: "Introuvable.",
  413: "Fichier trop volumineux (8 Mo maximum).",
  415: "Format non pris en charge (JPEG ou PNG uniquement).",
  429: "Doucement ! Trop de requêtes, réessaie dans un instant.",
  500: "Le serveur a rencontré un problème. Réessaie dans un instant."
};

function buildUrl(path: string, query?: Query): string {
  if (!API_BASE_URL) {
    throw new ApiError(0, "L'adresse du serveur n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const url = new URL(API_BASE_URL + path);
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null && v !== "") url.searchParams.set(k, String(v));
    }
  }
  return url.toString();
}

async function rawFetch(path: string, opts: RequestOptions, token: string | null): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  opts.signal?.addEventListener("abort", () => controller.abort(), { once: true });
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let body: BodyInit | undefined;
  if (opts.form) {
    body = opts.form;
  } else if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(opts.body);
  }
  try {
    return await fetch(buildUrl(path, opts.query), {
      method: opts.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (err) {
    if (err instanceof ApiError) throw err;
    if ((err as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "Le serveur met trop de temps à répondre.");
    }
    throw new ApiError(0, "Impossible de joindre le serveur. Vérifie ta connexion.");
  } finally {
    clearTimeout(timer);
  }
}

let refreshing: Promise<Tokens | null> | null = null;

/** Single-flight refresh: concurrent 401s share one refresh call (refresh tokens rotate). */
export function refreshTokens(): Promise<Tokens | null> {
  if (!refreshing) {
    refreshing = (async () => {
      const current = currentTokens();
      if (!current) return null;
      try {
        const res = await rawFetch(
          "/auth/refresh",
          { method: "POST", body: { refreshToken: current.refreshToken } },
          null
        );
        if (!res.ok) {
          if (res.status === 401) await clearTokens();
          return null;
        }
        const data = (await res.json()) as Tokens;
        const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
        await saveTokens(next);
        return next;
      } catch {
        return null;
      } finally {
        refreshing = null;
      }
    })();
  }
  return refreshing;
}

async function errorFrom(res: Response): Promise<ApiError> {
  let message: string | undefined;
  try {
    const data = (await res.json()) as { error?: string };
    message = data.error;
  } catch {
    message = undefined;
  }
  const friendly = FRIENDLY[res.status];
  // 4xx messages from the API are specific and already safe to show (e.g. "email already exists").
  return new ApiError(
    res.status,
    res.status === 400 || res.status === 409 || res.status === 403
      ? (message ?? friendly ?? "Requête invalide.")
      : (friendly ?? message ?? "Erreur inattendue.")
  );
}

export async function api<T = void>(path: string, opts: RequestOptions = {}): Promise<T> {
  const useAuth = opts.auth !== false;
  let res = await rawFetch(path, opts, useAuth ? (currentTokens()?.accessToken ?? null) : null);

  if (res.status === 401 && useAuth && currentTokens()) {
    const next = await refreshTokens();
    if (next) {
      res = await rawFetch(path, opts, next.accessToken);
    } else {
      sessionExpiredHandler?.();
    }
  }
  if (!res.ok) throw await errorFrom(res);
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

/** Source for expo-image: photos are private, so they are fetched with the bearer token. */
export function photoSource(path: string): {
  uri: string;
  headers: Record<string, string>;
  cacheKey: string;
} {
  const token = currentTokens()?.accessToken ?? "";
  return { uri: API_BASE_URL + path, headers: { Authorization: `Bearer ${token}` }, cacheKey: path };
}
