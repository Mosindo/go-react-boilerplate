import { Platform } from "react-native";
import { getFreshAccessToken, session } from "../auth/session";
import type { Tokens } from "../auth/tokenStore";

function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return explicit.replace(/\/+$/, "");
  if (Platform.OS === "web" && typeof window !== "undefined" && window.location?.hostname) {
    return `http://${window.location.hostname}:18080`;
  }
  // Android emulators reach the host machine through 10.0.2.2.
  return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
}

export const API_BASE_URL = resolveBaseUrl();

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public field?: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export function isApiError(err: unknown, status?: number): err is ApiError {
  return err instanceof ApiError && (status === undefined || err.status === status);
}

const TIMEOUT_MS = 15_000;

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
};

async function send(path: string, opts: RequestOptions, token: string | null): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  opts.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let body: BodyInit | undefined;
  if (opts.form) {
    body = opts.form; // fetch sets the multipart boundary itself
  } else if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(opts.body);
  }

  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      method: opts.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (err) {
    if ((err as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "Le serveur met trop de temps à répondre. Réessayez.");
    }
    throw new ApiError(0, "Connexion impossible. Vérifiez votre réseau.");
  } finally {
    clearTimeout(timer);
  }
}

async function parse<T>(res: Response): Promise<T> {
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  const data = text ? (JSON.parse(text) as unknown) : undefined;
  if (!res.ok) {
    const payload = (data ?? {}) as { error?: string; field?: string };
    throw new ApiError(res.status, payload.error ?? `Erreur ${res.status}`, payload.field);
  }
  return data as T;
}

/** Typed JSON request with automatic access-token refresh on 401. */
export async function api<T = void>(path: string, opts: RequestOptions = {}): Promise<T> {
  const authed = opts.auth !== false;
  let token = authed ? await getFreshAccessToken() : null;
  if (authed && !token) throw new ApiError(401, "Session expirée. Reconnectez-vous.");

  let res = await send(path, opts, token);
  if (res.status === 401 && authed) {
    const next = await session.refresh();
    if (!next) throw new ApiError(401, "Session expirée. Reconnectez-vous.");
    token = next.accessToken;
    res = await send(path, opts, token);
  }
  return parse<T>(res);
}

/** Used once at startup so the session layer can refresh without importing the client. */
export function installRefresh(): void {
  session.setRefreshImpl(async (refreshToken: string): Promise<Tokens> => {
    const res = await send("/auth/refresh", { method: "POST", body: { refreshToken } }, null);
    const data = await parse<{ accessToken: string; refreshToken: string }>(res);
    return { accessToken: data.accessToken, refreshToken: data.refreshToken };
  });
}
installRefresh();
