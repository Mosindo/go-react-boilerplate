import { Platform } from "react-native";
import { getTokens, saveTokens, type AuthTokens } from "../store/tokenStore";

function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return explicit.replace(/\/+$/, "");
  if (__DEV__) {
    // Android emulator reaches the host through 10.0.2.2. A physical phone
    // needs EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080 (see README).
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveBaseUrl();

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

const TIMEOUT_MS = 15_000;

type Options = {
  method?: string;
  body?: unknown;
  form?: FormData;
  signal?: AbortSignal;
  /** Skip auth header and the 401→refresh dance (login, register…). */
  anonymous?: boolean;
};

let refreshing: Promise<AuthTokens | null> | null = null;
let onSessionExpired: (() => void) | null = null;

export function setSessionExpiredHandler(fn: (() => void) | null): void {
  onSessionExpired = fn;
}

async function rawFetch(path: string, opts: Options, token: string | null): Promise<Response> {
  if (!API_BASE_URL) {
    throw new ApiError(0, "API non configurée (EXPO_PUBLIC_API_URL).");
  }
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  opts.signal?.addEventListener("abort", () => controller.abort(), { once: true });
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let body: BodyInit | undefined;
  if (opts.form) {
    body = opts.form;
  } else if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(opts.body);
  }
  try {
    return await fetch(`${API_BASE_URL}${path}`, {
      method: opts.method ?? (body ? "POST" : "GET"),
      headers,
      body,
      signal: controller.signal
    });
  } catch {
    throw new ApiError(0, "network error");
  } finally {
    clearTimeout(timer);
  }
}

/** Rotates the refresh token once even when several requests fail together. */
async function refreshTokens(): Promise<AuthTokens | null> {
  const current = getTokens();
  if (!current) return null;
  refreshing ??= (async () => {
    try {
      const res = await rawFetch(
        "/auth/refresh",
        { method: "POST", body: { refreshToken: current.refreshToken } },
        null
      );
      if (!res.ok) {
        if (res.status === 401 || res.status === 400) await saveTokens(null);
        return null;
      }
      const data = (await res.json()) as AuthTokens;
      const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
      await saveTokens(next);
      return next;
    } catch {
      return null;
    } finally {
      refreshing = null;
    }
  })();
  return refreshing;
}

async function parseError(res: Response): Promise<ApiError> {
  let message = `request failed (${res.status})`;
  try {
    const data = (await res.json()) as { error?: string };
    if (data.error) message = data.error;
  } catch {
    /* non-JSON error body */
  }
  return new ApiError(res.status, message);
}

export async function api<T = void>(path: string, opts: Options = {}): Promise<T> {
  let res = await rawFetch(path, opts, opts.anonymous ? null : (getTokens()?.accessToken ?? null));
  if (res.status === 401 && !opts.anonymous && getTokens()) {
    const next = await refreshTokens();
    if (next) {
      res = await rawFetch(path, opts, next.accessToken);
    } else {
      onSessionExpired?.();
    }
  }
  if (!res.ok) throw await parseError(res);
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
