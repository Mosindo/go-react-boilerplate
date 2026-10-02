import { API_URL } from "../config";

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }

  get isNetwork(): boolean {
    return this.status === 0;
  }
}

export interface Tokens {
  accessToken: string;
  refreshToken: string;
}

/** The auth layer plugs itself in here so the client stays free of UI/storage concerns. */
export interface SessionBridge {
  getTokens(): Tokens | null;
  setTokens(tokens: Tokens): void;
  onAuthLost(): void;
}

let bridge: SessionBridge | null = null;
let refreshing: Promise<Tokens | null> | null = null;

export function configureSession(b: SessionBridge | null): void {
  bridge = b;
}

type Body = Record<string, unknown> | unknown[] | FormData | undefined;

export interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: Body;
  /** false for login/register/refresh: no bearer token, no refresh-and-retry. */
  auth?: boolean;
  signal?: AbortSignal;
}

export function absoluteUrl(path: string): string {
  return /^https?:\/\//.test(path) ? path : `${API_URL}${path}`;
}

async function rawFetch(
  path: string,
  opts: RequestOptions,
  token: string | null,
): Promise<Response> {
  const headers: Record<string, string> = { Accept: "application/json" };
  let body: BodyInit | undefined;
  if (opts.body instanceof FormData) {
    body = opts.body; // fetch sets the multipart boundary itself
  } else if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(opts.body);
  }
  if (token) headers.Authorization = `Bearer ${token}`;
  try {
    return await fetch(absoluteUrl(path), {
      method: opts.method ?? "GET",
      headers,
      body,
      signal: opts.signal,
    });
  } catch (err) {
    if (err instanceof Error && err.name === "AbortError") throw err;
    throw new ApiError(0, "network", "Connexion impossible. Vérifiez votre réseau.");
  }
}

async function toError(res: Response): Promise<ApiError> {
  let code = "error";
  let message = "Une erreur est survenue.";
  try {
    const data = (await res.json()) as { error?: string; code?: string };
    if (data.code) code = data.code;
    if (data.error) message = data.error;
  } catch {
    // non-JSON error body: keep defaults
  }
  return new ApiError(res.status, code, message);
}

async function refreshTokens(): Promise<Tokens | null> {
  const current = bridge?.getTokens();
  if (!bridge || !current) return null;
  if (!refreshing) {
    refreshing = (async () => {
      try {
        const res = await rawFetch(
          "/auth/refresh",
          { method: "POST", body: { refreshToken: current.refreshToken }, auth: false },
          null,
        );
        if (res.status === 401) return null;
        if (!res.ok) throw await toError(res);
        const data = (await res.json()) as Tokens;
        const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
        bridge?.setTokens(next);
        return next;
      } finally {
        refreshing = null;
      }
    })();
  }
  return refreshing;
}

export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const useAuth = opts.auth !== false;
  let res = await rawFetch(path, opts, useAuth ? (bridge?.getTokens()?.accessToken ?? null) : null);

  if (res.status === 401 && useAuth && bridge?.getTokens()) {
    const next = await refreshTokens();
    if (!next) {
      bridge.onAuthLost();
      throw await toError(res);
    }
    res = await rawFetch(path, opts, next.accessToken);
  }
  if (!res.ok) throw await toError(res);
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}
