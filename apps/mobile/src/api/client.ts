import { Platform } from "react-native";
import { getAccessToken } from "../store/tokenStore";
import { showGlobalError } from "../shared/feedback";
import { refreshAccessToken } from "./session";

export type ApiErrorPayload = {
  error?: string;
  code?: string;
};

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function developmentFallbackBaseUrl(): string {
  return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
}

function resolveApiBaseUrl(): string {
  const explicitBaseUrl = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicitBaseUrl) {
    return normalizeBaseUrl(explicitBaseUrl);
  }

  if (__DEV__) {
    return developmentFallbackBaseUrl();
  }

  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

const DEFAULT_TIMEOUT_MS = 10_000;

export class ApiError extends Error {
  public status: number;
  public data: unknown;
  public code?: string;

  constructor(status: number, data: unknown, message?: string) {
    super(message ?? `API request failed (${status})`);
    this.name = "ApiError";
    this.status = status;
    this.data = data;
    this.code = (data as ApiErrorPayload | null)?.code;
  }
}

export type RequestOptions = RequestInit & {
  /** Overrides the 10s default (uploads need longer). */
  timeoutMs?: number;
  /** Set on the calls that must not trigger a session refresh. */
  skipRefresh?: boolean;
  /** Suppresses the global error banner; the caller renders its own message. */
  silent?: boolean;
};

function mergeHeaders(base: HeadersInit | undefined, token: string | null): Headers {
  const headers = new Headers();
  headers.set("Accept", "application/json");

  if (base) {
    new Headers(base).forEach((value, key) => {
      headers.set(key, value);
    });
  }
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  return headers;
}

function shouldSetJsonContentType(body: BodyInit | null | undefined): boolean {
  return body !== undefined && body !== null && !(body instanceof FormData);
}

function formatNetworkHint(): string {
  if (process.env.EXPO_PUBLIC_API_URL) {
    return "Network error. Check your connection and API availability, then try again.";
  }

  if (!__DEV__) {
    return "API base URL is not configured for this build. Set EXPO_PUBLIC_API_URL and try again.";
  }

  if (Platform.OS === "android") {
    return "Network error. Android emulators usually need 10.0.2.2, and physical devices need EXPO_PUBLIC_API_URL set to your LAN IP.";
  }

  if (Platform.OS === "ios") {
    return "Network error. iOS simulators can use localhost, but physical devices need EXPO_PUBLIC_API_URL set to your LAN IP.";
  }

  return "Network error. Check your connection and try again.";
}

function requireApiBaseUrl(): string {
  if (API_BASE_URL) {
    return API_BASE_URL;
  }

  const message = "API base URL is not configured. Set EXPO_PUBLIC_API_URL for this build.";
  showGlobalError(message);
  throw new Error(message);
}

/** Absolute URL for a server-relative asset path such as a photo. */
export function assetUrl(path: string): string {
  return `${API_BASE_URL}${path}`;
}

async function readErrorPayload(response: Response): Promise<unknown> {
  const contentType = response.headers.get("content-type") ?? "";
  try {
    return contentType.includes("application/json") ? await response.json() : await response.text();
  } catch {
    return null;
  }
}

async function send(path: string, options: RequestOptions): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), options.timeoutMs ?? DEFAULT_TIMEOUT_MS);
  try {
    if (options.signal) {
      if (options.signal.aborted) {
        controller.abort();
      } else {
        options.signal.addEventListener("abort", () => controller.abort(), { once: true });
      }
    }

    const headers = mergeHeaders(options.headers, await getAccessToken());
    if (!headers.has("Content-Type") && shouldSetJsonContentType(options.body)) {
      headers.set("Content-Type", "application/json");
    }
    return await fetch(`${requireApiBaseUrl()}${path}`, { ...options, headers, signal: controller.signal });
  } finally {
    clearTimeout(timeoutId);
  }
}

export async function apiRequest<T = unknown>(path: string, options: RequestOptions = {}): Promise<T> {
  try {
    let response = await send(path, options);

    if (response.status === 401 && !options.skipRefresh && !path.startsWith("/auth/")) {
      if (await refreshAccessToken()) {
        response = await send(path, options);
      }
    }

    if (!response.ok) {
      const data = await readErrorPayload(response);
      const apiPayload = data as ApiErrorPayload | null;
      const message =
        (typeof apiPayload === "object" && apiPayload?.error) ||
        (response.status === 401 ? "Your session is no longer valid. Please sign in again." : `Request failed (${response.status})`);
      if (response.status >= 500 && !options.silent) {
        showGlobalError(message);
      }
      throw new ApiError(response.status, data, message);
    }

    const text = await response.text();
    if (!text) {
      return undefined as T;
    }
    return JSON.parse(text) as T;
  } catch (err) {
    if (err instanceof ApiError) {
      throw err;
    }
    const name = (err as { name?: string }).name;
    if (name === "AbortError") {
      if (!options.silent) {
        showGlobalError("The request timed out. Please try again.");
      }
      throw new Error("The request timed out.", { cause: err });
    }
    if (err instanceof TypeError && !options.silent) {
      showGlobalError(formatNetworkHint());
    }
    throw err;
  }
}

export function errorMessage(error: unknown, fallback = "Something went wrong. Please try again."): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}
