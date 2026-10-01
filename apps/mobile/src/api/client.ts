import { Platform } from "react-native";
import { showGlobalError } from "../shared/feedback/store";
import { endpoints } from "./endpoints";
import type { AuthSession } from "./types";

export type AuthTokens = { accessToken: string; refreshToken: string };

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }
  if (__DEV__) {
    // Android emulators reach the host through 10.0.2.2; physical phones need EXPO_PUBLIC_API_URL.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export class ApiError extends Error {
  public readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

/** Storage + session hooks injected by AuthProvider so the client stays UI-free. */
export type AuthBridge = {
  getTokens: () => Promise<AuthTokens | null>;
  setTokens: (tokens: AuthTokens, user: AuthSession["user"]) => Promise<void>;
  onAuthLost: () => void;
};

let bridge: AuthBridge | null = null;
let refreshInFlight: Promise<string | null> | null = null;

export function setAuthBridge(next: AuthBridge | null): void {
  bridge = next;
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  form?: FormData;
  query?: Record<string, string | number | undefined>;
  /** Set false for endpoints that must not carry (or trigger a refresh of) a token. */
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

const NETWORK_MESSAGE = "Connexion impossible. Vérifiez votre réseau puis réessayez.";

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  if (!API_BASE_URL) {
    throw new ApiError(0, "L'adresse de l'API n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const params = Object.entries(query ?? {})
    .filter((entry): entry is [string, string | number] => entry[1] !== undefined)
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
    .join("&");
  return `${API_BASE_URL}${path}${params ? `?${params}` : ""}`;
}

async function send(path: string, options: RequestOptions, token: string | null): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form; // fetch sets the multipart boundary itself
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  try {
    return await fetch(buildUrl(path, options.query), { method: options.method ?? "GET", headers, body, signal: controller.signal });
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    const aborted = (error as { name?: string }).name === "AbortError";
    if (options.signal?.aborted) {
      throw error;
    }
    const message = aborted ? "Le serveur met trop de temps à répondre." : NETWORK_MESSAGE;
    showGlobalError(message);
    throw new ApiError(0, message);
  } finally {
    clearTimeout(timer);
  }
}

async function readError(response: Response): Promise<ApiError> {
  let message = `Erreur ${response.status}`;
  try {
    const data = (await response.json()) as { error?: string };
    if (data?.error) {
      message = data.error;
    }
  } catch {
    // body was not JSON; keep the generic message
  }
  if (response.status >= 500) {
    message = "Le service rencontre un problème. Réessayez dans un instant.";
    showGlobalError(message);
  } else if (response.status === 429) {
    message = "Trop de requêtes. Patientez un instant.";
    showGlobalError(message);
  }
  return new ApiError(response.status, message);
}

/** One refresh at a time: concurrent 401s share the same promise. */
async function refreshAccessToken(): Promise<string | null> {
  if (!bridge) {
    return null;
  }
  refreshInFlight ??= (async () => {
    const tokens = await bridge?.getTokens();
    if (!tokens?.refreshToken) {
      return null;
    }
    const response = await send(endpoints.auth.refresh, { method: "POST", body: { refreshToken: tokens.refreshToken }, auth: false }, null);
    if (!response.ok) {
      if (response.status === 401 || response.status === 403) {
        bridge?.onAuthLost();
      }
      return null;
    }
    const session = (await response.json()) as AuthSession;
    await bridge?.setTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken }, session.user);
    return session.accessToken;
  })().finally(() => {
    refreshInFlight = null;
  });
  return refreshInFlight;
}

export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  const tokens = useAuth ? await bridge?.getTokens() : null;
  let response = await send(path, options, tokens?.accessToken ?? null);

  if (response.status === 401 && useAuth && tokens?.refreshToken) {
    const fresh = await refreshAccessToken();
    if (fresh) {
      response = await send(path, options, fresh);
    }
  }

  if (!response.ok) {
    throw await readError(response);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
