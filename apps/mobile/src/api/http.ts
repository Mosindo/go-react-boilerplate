import { API_BASE_URL } from "./config";
import { clearTokens, loadTokens, saveTokens, type Tokens } from "../auth/tokenStorage";
import type { AuthResponse } from "./types";

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

const FRIENDLY: Record<string, string> = {
  invalid_credentials: "E-mail ou mot de passe incorrect.",
  email_exists: "Un compte existe déjà avec cet e-mail.",
  rate_limited: "Trop de tentatives. Patientez un instant avant de réessayer.",
  profile_incomplete: "Complétez votre profil (au moins une photo) pour continuer.",
  already_decided: "Vous avez déjà réagi à ce profil.",
  file_too_large: "Cette photo est trop lourde (5 Mo maximum).",
  invalid_reset_token: "Ce code est invalide ou a expiré.",
};

export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    return (
      FRIENDLY[err.code] ??
      (err.status >= 500 ? "Un problème est survenu de notre côté. Réessayez." : err.message)
    );
  }
  return "Connexion impossible. Vérifiez votre réseau et réessayez.";
}

type SessionListener = {
  onTokens: (tokens: Tokens) => void;
  onSignedOut: () => void;
};
let listener: SessionListener | null = null;
export function setSessionListener(l: SessionListener | null): void {
  listener = l;
}

let memoryTokens: Tokens | null = null;
let loaded = false;
let refreshing: Promise<Tokens | null> | null = null;

export async function getTokens(): Promise<Tokens | null> {
  if (!loaded) {
    memoryTokens = await loadTokens();
    loaded = true;
  }
  return memoryTokens;
}

export async function setTokens(tokens: Tokens | null): Promise<void> {
  memoryTokens = tokens;
  loaded = true;
  if (tokens) await saveTokens(tokens);
  else await clearTokens();
}

async function timedFetch(path: string, init: RequestInit): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 20_000);
  try {
    return await fetch(API_BASE_URL + path, { ...init, signal: controller.signal });
  } finally {
    clearTimeout(timer);
  }
}

async function parseError(res: Response): Promise<ApiError> {
  let code = "http_error";
  let message = `Erreur ${res.status}`;
  try {
    const data = (await res.json()) as { error?: string; code?: string };
    code = data.code ?? code;
    message = data.error ?? message;
  } catch {
    // non-JSON error body
  }
  return new ApiError(res.status, code, message);
}

/** Single-flight refresh: concurrent 401s share one refresh call (refresh tokens rotate). */
async function refreshTokens(): Promise<Tokens | null> {
  if (!refreshing) {
    refreshing = (async () => {
      const current = await getTokens();
      if (!current) return null;
      try {
        const res = await timedFetch("/auth/refresh", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refreshToken: current.refreshToken }),
        });
        if (!res.ok) {
          if (res.status === 401) {
            await setTokens(null);
            listener?.onSignedOut();
          }
          return null;
        }
        const data = (await res.json()) as AuthResponse;
        const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
        await setTokens(next);
        listener?.onTokens(next);
        return next;
      } catch {
        return null;
      }
    })().finally(() => {
      refreshing = null;
    });
  }
  return refreshing;
}

export async function getFreshAccessToken(): Promise<string | null> {
  const tokens = await refreshTokens();
  return tokens?.accessToken ?? null;
}

type RequestOptions = { method?: string; body?: unknown; form?: FormData; auth?: boolean };

export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, form, auth = true } = opts;

  const send = async (token: string | null): Promise<Response> => {
    const headers: Record<string, string> = { Accept: "application/json" };
    if (token) headers.Authorization = `Bearer ${token}`;
    let payload: BodyInit | undefined;
    if (form) payload = form;
    else if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      payload = JSON.stringify(body);
    }
    return timedFetch(path, { method, headers, body: payload });
  };

  let token = auth ? ((await getTokens())?.accessToken ?? null) : null;
  let res = await send(token);
  if (res.status === 401 && auth) {
    token = await getFreshAccessToken();
    if (token) res = await send(token);
  }
  if (!res.ok) throw await parseError(res);
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}
