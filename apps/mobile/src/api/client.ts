import { Platform } from "react-native";
import type { AuthResponse } from "./types";

function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return explicit.replace(/\/+$/, "");
  if (__DEV__)
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  return "";
}

export const API_BASE_URL = resolveBaseUrl();

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }

  get isNetwork(): boolean {
    return this.status === 0;
  }
}

/** The auth layer plugs itself in here so the HTTP client never depends on React. */
export type SessionBridge = {
  getRefreshToken: () => Promise<string | null>;
  onTokens: (auth: AuthResponse) => void | Promise<void>;
  onExpired: () => void | Promise<void>;
};

let bridge: SessionBridge | null = null;
let accessToken: string | null = null;
let accessExpiresAt = 0;
let refreshing: Promise<boolean> | null = null;

export function configureSession(next: SessionBridge | null): void {
  bridge = next;
}

export function setAccessToken(token: string | null): void {
  accessToken = token;
  accessExpiresAt = token ? readExpiry(token) : 0;
}

export function readExpiry(jwt: string): number {
  try {
    const payload = jwt.split(".")[1];
    if (!payload) return 0;
    const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
    const exp = (JSON.parse(json) as { exp?: number }).exp;
    return typeof exp === "number" ? exp * 1000 : 0;
  } catch {
    return 0;
  }
}

type RequestOptions = {
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
};

async function parse(res: Response): Promise<unknown> {
  const text = await res.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

async function send(method: string, path: string, opts: RequestOptions): Promise<Response> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (opts.auth !== false && accessToken) headers.Authorization = `Bearer ${accessToken}`;
  let body: BodyInit | undefined;
  if (opts.form) body = opts.form;
  else if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(opts.body);
  }
  try {
    return await fetch(`${API_BASE_URL}${path}`, { method, headers, body, signal: opts.signal });
  } catch {
    throw new ApiError(0, "Connexion impossible. Vérifiez votre réseau et réessayez.");
  }
}

/** Single-flight refresh: concurrent 401s share one rotation of the refresh token. */
export function refreshSession(): Promise<boolean> {
  if (!refreshing) {
    refreshing = (async () => {
      try {
        const refreshToken = await bridge?.getRefreshToken();
        if (!bridge || !refreshToken) return false;
        const res = await send("POST", "/auth/refresh", { body: { refreshToken }, auth: false });
        if (!res.ok) {
          // Only an explicit rejection ends the session; a server hiccup must not log people out.
          if (res.status === 401 || res.status === 403) await bridge.onExpired();
          return false;
        }
        const auth = (await parse(res)) as AuthResponse;
        setAccessToken(auth.accessToken);
        await bridge.onTokens(auth);
        return true;
      } catch {
        return false;
      } finally {
        refreshing = null;
      }
    })();
  }
  return refreshing;
}

/** Returns a non-expired access token, refreshing first when it is about to expire. */
export async function getFreshAccessToken(): Promise<string | null> {
  if (accessToken && accessExpiresAt - Date.now() > 30_000) return accessToken;
  if (await refreshSession()) return accessToken;
  return null;
}

const MESSAGES: Record<number, string> = {
  400: "Requête invalide.",
  401: "Votre session a expiré.",
  403: "Action non autorisée.",
  404: "Élément introuvable.",
  409: "Cette action est impossible dans l'état actuel.",
  413: "Fichier trop volumineux.",
  415: "Format de fichier non pris en charge (JPEG, PNG ou WebP).",
  422: "Certaines informations sont invalides.",
  429: "Trop de tentatives. Patientez un instant.",
};

export async function request<T>(
  method: string,
  path: string,
  opts: RequestOptions = {},
): Promise<T> {
  let res = await send(method, path, opts);
  if (res.status === 401 && opts.auth !== false && (await refreshSession())) {
    res = await send(method, path, opts);
  }
  const data = await parse(res);
  if (!res.ok) {
    const serverMessage =
      data && typeof data === "object" && "error" in data
        ? String((data as { error: unknown }).error)
        : "";
    // Server messages are written for end users on 4xx; 5xx stays generic.
    const message =
      res.status >= 500
        ? "Le service est momentanément indisponible. Réessayez plus tard."
        : serverMessage || MESSAGES[res.status] || "Une erreur est survenue.";
    throw new ApiError(res.status, message);
  }
  return data as T;
}

export const api = {
  get: <T>(path: string, signal?: AbortSignal) => request<T>("GET", path, { signal }),
  post: <T>(path: string, body?: unknown) => request<T>("POST", path, { body }),
  put: <T>(path: string, body?: unknown) => request<T>("PUT", path, { body }),
  del: <T>(path: string, body?: unknown) => request<T>("DELETE", path, { body }),
  upload: <T>(method: "POST" | "PUT", path: string, form: FormData) =>
    request<T>(method, path, { form }),
  public: <T>(method: string, path: string, body?: unknown) =>
    request<T>(method, path, { body, auth: false }),
};
