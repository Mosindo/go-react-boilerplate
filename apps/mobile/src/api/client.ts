import { Platform } from "react-native";
import { clearTokens, getTokens, saveTokens, type AuthTokens } from "../store/tokenStore";
import type { AuthResponse } from "./types";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return normalizeBaseUrl(explicit);
  if (__DEV__) return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** Photo URLs are returned relative to the API; make them loadable. */
export function absoluteUrl(path: string): string {
  return /^https?:\/\//.test(path) ? path : `${API_BASE_URL}${path}`;
}

export function websocketUrl(): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws`;
}

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

const FRIENDLY: Record<string, string> = {
  invalid_credentials: "E-mail ou mot de passe incorrect.",
  email_taken: "Un compte existe déjà avec cet e-mail.",
  underage: "Vous devez avoir 18 ans ou plus pour utiliser Alma.",
  rate_limited: "Trop de tentatives. Patientez un instant avant de réessayer.",
  invalid_code: "Ce code est invalide ou a expiré.",
  wrong_password: "Mot de passe incorrect.",
  photo_limit: "Vous pouvez ajouter 6 photos au maximum.",
  unsupported_type: "Format non pris en charge (JPEG, PNG ou WebP).",
  file_too_large: "Cette photo est trop lourde (8 Mo maximum).",
  account_suspended: "Ce compte a été suspendu."
};

type Listener = () => void;
let sessionExpiredListener: Listener | null = null;
/** Called once when the refresh token is rejected and the session cannot be recovered. */
export function onSessionExpired(listener: Listener | null): void {
  sessionExpiredListener = listener;
}

type Options = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  query?: Record<string, string | number | undefined | null>;
  auth?: boolean;
  signal?: AbortSignal;
};

function buildUrl(path: string, query?: Options["query"]): string {
  if (!API_BASE_URL) throw new ApiError(0, "no_api_url", "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  const params = Object.entries(query ?? {})
    .filter(([, v]) => v !== undefined && v !== null && v !== "")
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
    .join("&");
  return `${API_BASE_URL}${path}${params ? `?${params}` : ""}`;
}

async function send(path: string, opts: Options, token: string | null): Promise<Response> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let body: BodyInit | undefined;
  if (opts.form) body = opts.form;
  else if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(opts.body);
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), opts.form ? 60_000 : 15_000);
  opts.signal?.addEventListener("abort", () => controller.abort(), { once: true });
  try {
    return await fetch(buildUrl(path, opts.query), { method: opts.method ?? "GET", headers, body, signal: controller.signal });
  } catch (err) {
    if (err instanceof ApiError) throw err;
    const aborted = (err as { name?: string }).name === "AbortError";
    throw new ApiError(
      0,
      aborted ? "timeout" : "network",
      aborted ? "Le serveur met trop de temps à répondre." : "Connexion impossible. Vérifiez votre réseau et réessayez."
    );
  } finally {
    clearTimeout(timeout);
  }
}

let refreshing: Promise<AuthTokens | null> | null = null;

/** Single-flight token refresh shared by every concurrent request. */
export function refreshTokens(): Promise<AuthTokens | null> {
  if (!refreshing) {
    refreshing = (async () => {
      const current = await getTokens();
      if (!current) return null;
      try {
        const res = await send("/auth/refresh", { method: "POST", body: { refreshToken: current.refreshToken } }, null);
        if (!res.ok) {
          // Only a definitive rejection ends the session; outages keep the tokens.
          if (res.status === 401 || res.status === 400) {
            await clearTokens();
            sessionExpiredListener?.();
          }
          return null;
        }
        const data = (await res.json()) as AuthResponse;
        const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
        await saveTokens(next);
        return next;
      } catch {
        return null;
      } finally {
        setTimeout(() => {
          refreshing = null;
        }, 0);
      }
    })();
  }
  return refreshing;
}

function jwtExpiry(token: string): number | null {
  try {
    const payload = token.split(".")[1];
    if (!payload) return null;
    const json = JSON.parse(atob(payload.replace(/-/g, "+").replace(/_/g, "/"))) as { exp?: number };
    return typeof json.exp === "number" ? json.exp * 1000 : null;
  } catch {
    return null;
  }
}

/** Returns an access token valid for at least 30 s, refreshing proactively. */
export async function getValidAccessToken(): Promise<string | null> {
  const tokens = await getTokens();
  if (!tokens) return null;
  const exp = jwtExpiry(tokens.accessToken);
  if (exp !== null && exp - Date.now() < 30_000) {
    return (await refreshTokens())?.accessToken ?? null;
  }
  return tokens.accessToken;
}

async function toError(res: Response): Promise<ApiError> {
  let code = "http_error";
  let message = `Une erreur est survenue (${res.status}).`;
  try {
    const data = (await res.json()) as { error?: string; code?: string };
    if (data.code) code = data.code;
    if (data.error) message = data.error;
  } catch {
    // non-JSON error body
  }
  return new ApiError(res.status, code, FRIENDLY[code] ?? message);
}

export async function apiRequest<T = void>(path: string, opts: Options = {}): Promise<T> {
  const needsAuth = opts.auth !== false;
  let token = needsAuth ? await getValidAccessToken() : null;
  let res = await send(path, opts, token);

  if (res.status === 401 && needsAuth) {
    const next = await refreshTokens();
    if (next) {
      token = next.accessToken;
      res = await send(path, opts, token);
    }
  }
  if (!res.ok) throw await toError(res);
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
