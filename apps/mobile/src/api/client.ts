import { API_BASE_URL } from "../config";
import { clearTokens, getTokens, setTokens } from "../auth/session";
import type { AuthResponse } from "./types";

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

type Query = Record<string, string | number | undefined | null>;

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  query?: Query;
  auth?: boolean;
  timeoutMs?: number;
  signal?: AbortSignal;
};

type FetchLike = typeof fetch;
let fetchImpl: FetchLike = (...args) => fetch(...args);

/** Test hook: replace the network layer. */
export function setFetchImplementation(impl: FetchLike) {
  fetchImpl = impl;
}

function buildUrl(path: string, query?: Query): string {
  if (!API_BASE_URL) {
    throw new ApiError(0, "missing_api_url", "EXPO_PUBLIC_API_URL is not configured for this build.");
  }
  const params = Object.entries(query ?? {})
    .filter(([, value]) => value !== undefined && value !== null && value !== "")
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`);
  return `${API_BASE_URL}${path}${params.length ? `?${params.join("&")}` : ""}`;
}

async function parseError(response: Response): Promise<ApiError> {
  let code = "http_error";
  let message = `Request failed (${response.status})`;
  try {
    const payload = (await response.json()) as { error?: string; code?: string };
    if (payload.code) code = payload.code;
    if (payload.error) message = payload.error;
  } catch {
    // non-JSON error body
  }
  return new ApiError(response.status, code, message);
}

async function send(path: string, options: RequestOptions, accessToken: string | null): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? 15_000);
  const forwardAbort = () => controller.abort();
  options.signal?.addEventListener("abort", forwardAbort, { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`;
  }

  try {
    return await fetchImpl(buildUrl(path, options.query), {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (error) {
    if (error instanceof ApiError) throw error;
    if ((error as { name?: string }).name === "AbortError") {
      if (options.signal?.aborted) throw error;
      throw new ApiError(0, "timeout", "The request timed out.");
    }
    throw new ApiError(0, "network", "Network unavailable.");
  } finally {
    clearTimeout(timeout);
    options.signal?.removeEventListener("abort", forwardAbort);
  }
}

let refreshInFlight: Promise<boolean> | null = null;

/** Rotates the refresh token once, even when many requests expire together. */
export function refreshAccessToken(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const tokens = getTokens();
      if (!tokens) return false;
      try {
        const response = await send("/auth/refresh", { method: "POST", body: { refreshToken: tokens.refreshToken } }, null);
        if (!response.ok) {
          if (response.status === 401) {
            await clearTokens();
          }
          return false;
        }
        const payload = (await response.json()) as AuthResponse;
        await setTokens({ accessToken: payload.accessToken, refreshToken: payload.refreshToken });
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

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  let response = await send(path, options, useAuth ? getTokens()?.accessToken ?? null : null);

  if (useAuth && response.status === 401 && getTokens()) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      response = await send(path, options, getTokens()?.accessToken ?? null);
      if (response.status === 401) {
        await clearTokens();
      }
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
