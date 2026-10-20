import { Platform } from "react-native";
import { clearTokens, getTokens, saveTokens } from "../store/tokenStore";
import { normalizeBaseUrl } from "../lib/url";
import type { AuthSession } from "./types";

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return normalizeBaseUrl(explicit);
  if (__DEV__) {
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export class ApiError extends Error {
  public status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
};

const REQUEST_TIMEOUT_MS = 15_000;

let authLostHandler: (() => void) | null = null;
let refreshInFlight: Promise<string | null> | null = null;

export function setAuthLostHandler(handler: (() => void) | null): void {
  authLostHandler = handler;
}

async function rawFetch(
  path: string,
  options: RequestOptions,
  accessToken: string | null
): Promise<Response> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "The API address is not configured. Set EXPO_PUBLIC_API_URL for this build.");
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`;
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (error) {
    if ((error as { name?: string }).name === "AbortError") {
      throw new ApiError(0, "The request timed out. Please try again.");
    }
    throw new ApiError(0, "Network error. Check your connection and try again.");
  } finally {
    clearTimeout(timeout);
  }
}

async function refreshAccessToken(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = await getTokens();
      if (!tokens?.refreshToken) return null;
      try {
        const response = await rawFetch(
          "/auth/refresh",
          { method: "POST", body: { refreshToken: tokens.refreshToken } },
          null
        );
        if (!response.ok) {
          await clearTokens();
          return null;
        }
        const session = (await response.json()) as AuthSession;
        await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
        return session.accessToken;
      } catch {
        // Transient failure: keep tokens so a later request can retry.
        return null;
      }
    })().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

async function toApiError(response: Response): Promise<ApiError> {
  let message = `Request failed (${response.status})`;
  try {
    const data = (await response.json()) as { error?: string };
    if (data.error) message = data.error;
  } catch {
    // Non-JSON error body.
  }
  return new ApiError(response.status, message);
}

export async function apiRequest<T = void>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let token = useAuth ? ((await getTokens())?.accessToken ?? null) : null;
  let response = await rawFetch(path, options, token);

  if (response.status === 401 && useAuth) {
    token = await refreshAccessToken();
    if (token) {
      response = await rawFetch(path, options, token);
    }
    if (response.status === 401 || !token) {
      const stillHasTokens = await getTokens();
      if (!stillHasTokens) authLostHandler?.();
    }
  }

  if (!response.ok) throw await toApiError(response);
  if (response.status === 204) return undefined as T;
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export async function currentAccessToken(): Promise<string | null> {
  return (await getTokens())?.accessToken ?? null;
}
