import { API_BASE_URL } from "./config";
import type { AuthResponse } from "./types";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";

const REQUEST_TIMEOUT_MS = 15_000;

export class ApiError extends Error {
  public readonly status: number;
  public readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

type SessionListener = () => void;
let onSessionExpired: SessionListener | null = null;

export function setSessionExpiredListener(listener: SessionListener | null): void {
  onSessionExpired = listener;
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  query?: Record<string, string | number | undefined | null>;
  auth?: boolean;
  signal?: AbortSignal;
};

export function buildQuery(query: RequestOptions["query"]): string {
  if (!query) {
    return "";
  }
  const parts: string[] = [];
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null && value !== "") {
      parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`);
    }
  }
  return parts.length ? `?${parts.join("&")}` : "";
}

async function toApiError(response: Response): Promise<ApiError> {
  let code = "unknown";
  let message = `Request failed (${response.status})`;
  try {
    const data = (await response.json()) as { error?: string; code?: string };
    if (data.error) {
      message = data.error;
    }
    if (data.code) {
      code = data.code;
    }
  } catch {
    // Non JSON error body.
  }
  return new ApiError(response.status, code, message);
}

async function rawFetch(path: string, options: RequestOptions, accessToken: string | null): Promise<Response> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "no_api_url", "API URL is not configured (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`;
  }
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  try {
    return await fetch(`${API_BASE_URL}${path}${buildQuery(options.query)}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (error) {
    if ((error as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "timeout", "The request timed out. Please try again.");
    }
    throw new ApiError(0, "network", "Network error. Check your connection and try again.");
  } finally {
    clearTimeout(timeout);
  }
}

let refreshInFlight: Promise<string | null> | null = null;

/** Single-flight refresh: concurrent 401s share one refresh call. */
export function refreshAccessToken(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = await getTokens();
      if (!tokens?.refreshToken) {
        return null;
      }
      try {
        const response = await rawFetch(
          "/auth/refresh",
          { method: "POST", body: { refreshToken: tokens.refreshToken } },
          null
        );
        if (!response.ok) {
          if (response.status === 401 || response.status === 400) {
            await clearTokens();
            onSessionExpired?.();
          }
          return null;
        }
        const session = (await response.json()) as AuthResponse;
        await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
        return session.accessToken;
      } catch {
        return null;
      }
    })().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  const tokens = useAuth ? await getTokens() : null;
  let response = await rawFetch(path, options, tokens?.accessToken ?? null);

  if (response.status === 401 && useAuth && tokens?.refreshToken) {
    const fresh = await refreshAccessToken();
    if (fresh) {
      response = await rawFetch(path, options, fresh);
    }
  }

  if (!response.ok) {
    throw await toApiError(response);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
