import { Platform } from "react-native";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import { endpoints } from "./endpoints";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

// The backend publishes port 18080 (see docker-compose.yml). A physical phone
// cannot reach "localhost": set EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080.
function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }
  if (__DEV__) {
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** Resolves a relative API path (such as a signed photo URL) to an absolute URL. */
export function mediaUrl(path: string | undefined | null): string | undefined {
  if (!path) {
    return undefined;
  }
  return /^https?:\/\//.test(path) ? path : `${API_BASE_URL}${path}`;
}

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

export type FileUpload = { uri: string; name: string; type: string };

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  query?: Record<string, string | number | undefined>;
  file?: FileUpload;
  /** Set to false for endpoints that must not carry credentials or trigger a refresh (login, register...). */
  auth?: boolean;
  timeoutMs?: number;
  signal?: AbortSignal;
};

type AuthResult = { accessToken: string; refreshToken: string };

let onSessionExpired: (() => void) | null = null;
let refreshInFlight: Promise<boolean> | null = null;

/** The auth provider registers here to sign the user out when refreshing fails. */
export function setSessionExpiredHandler(handler: (() => void) | null): void {
  onSessionExpired = handler;
}

async function rawFetch(path: string, options: RequestOptions, accessToken: string | null): Promise<Response> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "not_configured", "API base URL is not configured");
  }

  const query = options.query
    ? Object.entries(options.query)
        .filter((entry): entry is [string, string | number] => entry[1] !== undefined && entry[1] !== "")
        .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
        .join("&")
    : "";

  const headers: Record<string, string> = { Accept: "application/json" };
  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`;
  }

  let body: BodyInit | undefined;
  if (options.file) {
    const form = new FormData();
    // React Native's FormData accepts a { uri, name, type } descriptor in place of a Blob.
    form.append("file", options.file as unknown as Blob);
    body = form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? (options.file ? 60_000 : 15_000));
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  try {
    return await fetch(`${API_BASE_URL}${path}${query ? `?${query}` : ""}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (error) {
    const aborted = (error as { name?: string }).name === "AbortError";
    throw new ApiError(0, aborted ? "timeout" : "network", aborted ? "request timed out" : "network error");
  } finally {
    clearTimeout(timeout);
  }
}

async function parseError(response: Response): Promise<ApiError> {
  let code = "unknown";
  let message = `request failed (${response.status})`;
  try {
    const payload = (await response.json()) as { error?: string; code?: string };
    code = payload.code ?? code;
    message = payload.error ?? message;
  } catch {
    // Non-JSON error body (proxy error page, ...): keep the generic message.
  }
  return new ApiError(response.status, code, message);
}

/** Exchanges the refresh token for a new pair. Concurrent callers share one request. */
export function refreshSession(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = await getTokens();
      if (!tokens?.refreshToken) {
        return false;
      }
      try {
        const response = await rawFetch(endpoints.auth.refresh, { method: "POST", body: { refreshToken: tokens.refreshToken } }, null);
        if (!response.ok) {
          return false;
        }
        const next = (await response.json()) as AuthResult;
        await saveTokens({ accessToken: next.accessToken, refreshToken: next.refreshToken });
        return true;
      } catch {
        return false;
      }
    })().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let tokens = useAuth ? await getTokens() : null;
  let response = await rawFetch(path, options, tokens?.accessToken ?? null);

  if (response.status === 401 && useAuth && tokens) {
    if (await refreshSession()) {
      tokens = await getTokens();
      response = await rawFetch(path, options, tokens?.accessToken ?? null);
    }
    if (response.status === 401) {
      await clearTokens();
      onSessionExpired?.();
    }
  }

  if (!response.ok) {
    throw await parseError(response);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
