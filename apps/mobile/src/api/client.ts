import { Platform } from "react-native";
import { clearTokens, currentTokens, loadTokens, saveTokens } from "./tokenStore";
import type { AuthResponse } from "./types";

function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return explicit.replace(/\/+$/, "");
  if (__DEV__) {
    if (Platform.OS === "web" && typeof location !== "undefined") {
      return `${location.protocol}//${location.hostname}:18080`;
    }
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveBaseUrl();

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
  get isNetwork(): boolean {
    return this.status === 0;
  }
}

type SessionListener = () => void;
let onSessionExpired: SessionListener = () => undefined;
export function setSessionExpiredHandler(listener: SessionListener): void {
  onSessionExpired = listener;
}

let refreshing: Promise<boolean> | null = null;

/** Exchanges the refresh token for a new pair. Concurrent callers share one request. */
export function refreshSession(): Promise<boolean> {
  if (refreshing) return refreshing;
  refreshing = (async () => {
    const tokens = currentTokens() ?? (await loadTokens());
    if (!tokens?.refreshToken) return false;
    try {
      const res = await fetch(`${API_BASE_URL}/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ refreshToken: tokens.refreshToken })
      });
      if (!res.ok) {
        // Only a definitive rejection ends the session; transient errors keep it.
        if (res.status === 401 || res.status === 400) await clearTokens();
        return false;
      }
      const data = (await res.json()) as AuthResponse;
      await saveTokens({ accessToken: data.accessToken, refreshToken: data.refreshToken });
      return true;
    } catch {
      return false;
    }
  })().finally(() => {
    refreshing = null;
  });
  return refreshing;
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  query?: Record<string, string | number | undefined>;
  signal?: AbortSignal;
};

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  const url = `${API_BASE_URL}${path}`;
  if (!query) return url;
  const params = Object.entries(query)
    .filter(
      (entry): entry is [string, string | number] => entry[1] !== undefined && entry[1] !== ""
    )
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
    .join("&");
  return params ? `${url}?${params}` : url;
}

async function send(path: string, opts: RequestOptions, token: string | null): Promise<Response> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (opts.body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;
  try {
    return await fetch(buildUrl(path, opts.query), {
      method: opts.method ?? "GET",
      headers,
      body: opts.form ?? (opts.body !== undefined ? JSON.stringify(opts.body) : undefined),
      signal: opts.signal
    });
  } catch {
    throw new ApiError(0, "network_error", "network error");
  }
}

async function toError(res: Response): Promise<ApiError> {
  let code = "http_error";
  let message = `request failed (${res.status})`;
  try {
    const payload = (await res.json()) as { error?: string; code?: string };
    if (payload.code) code = payload.code;
    if (payload.error) message = payload.error;
  } catch {
    // Non-JSON error body.
  }
  return new ApiError(res.status, code, message);
}

/** Performs an authenticated request, transparently refreshing an expired access token once. */
export async function request(path: string, opts: RequestOptions = {}): Promise<Response> {
  const useAuth = opts.auth !== false;
  let token = useAuth ? ((currentTokens() ?? (await loadTokens()))?.accessToken ?? null) : null;
  let res = await send(path, opts, token);
  if (useAuth && res.status === 401) {
    if (await refreshSession()) {
      token = currentTokens()?.accessToken ?? null;
      res = await send(path, opts, token);
    }
    if (res.status === 401) {
      if (!currentTokens()) onSessionExpired();
      throw await toError(res);
    }
  }
  if (!res.ok) throw await toError(res);
  return res;
}

export async function api<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const res = await request(path, opts);
  // 202/204 answers carry no body.
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
