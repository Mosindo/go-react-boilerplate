import { API_BASE_URL } from "../config/env";
import { tokenStore, type AuthTokens, type TokenStore } from "../store/tokenStore";
import type { AuthResponse } from "./types";

/** HTTP error returned by the API. `code` is the stable machine-readable reason (e.g. "already_swiped"). */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly retryAfterSeconds?: number
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/** The request never reached the API (offline, DNS, timeout, wrong URL). */
export class NetworkError extends Error {
  constructor(message = "Impossible de joindre le serveur. Vérifiez votre connexion.") {
    super(message);
    this.name = "NetworkError";
  }
}

export type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  form?: FormData;
  auth?: boolean;
  signal?: AbortSignal;
  timeoutMs?: number;
};

type ClientDeps = {
  baseUrl: string;
  tokens: Pick<TokenStore, "load" | "save" | "clear">;
  fetchImpl?: typeof fetch;
  onSessionExpired?: () => void;
};

const DEFAULT_TIMEOUT_MS = 15_000;
const UPLOAD_TIMEOUT_MS = 60_000;

export class ApiClient {
  private refreshing: Promise<AuthTokens | null> | null = null;
  private onSessionExpired?: () => void;

  constructor(private readonly deps: ClientDeps) {
    this.onSessionExpired = deps.onSessionExpired;
  }

  get baseUrl(): string {
    return this.deps.baseUrl;
  }

  setSessionExpiredHandler(handler: () => void): void {
    this.onSessionExpired = handler;
  }

  /** Turns an API-relative media path (signed photo URL) into an absolute URL. */
  mediaUrl(path: string): string {
    return /^https?:\/\//.test(path) ? path : `${this.deps.baseUrl}${path}`;
  }

  async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const auth = options.auth ?? true;
    let tokens = auth ? await this.deps.tokens.load() : null;
    if (auth && !tokens) {
      throw new ApiError(401, "unauthorized", "Session expirée. Reconnectez-vous.");
    }

    let response = await this.send(path, options, tokens?.accessToken);
    if (response.status === 401 && auth) {
      // The access token is short-lived: refresh once, then replay the request.
      tokens = await this.refreshTokens(tokens?.refreshToken);
      if (!tokens) {
        this.onSessionExpired?.();
        throw new ApiError(401, "unauthorized", "Session expirée. Reconnectez-vous.");
      }
      response = await this.send(path, options, tokens.accessToken);
    }
    return this.parse<T>(response);
  }

  /** Forces a token refresh (used by the realtime connection before it reconnects). */
  async ensureFreshAccessToken(): Promise<string | null> {
    const current = await this.deps.tokens.load();
    const refreshed = await this.refreshTokens(current?.refreshToken);
    return refreshed?.accessToken ?? null;
  }

  async currentAccessToken(): Promise<string | null> {
    return (await this.deps.tokens.load())?.accessToken ?? null;
  }

  private async send(path: string, options: RequestOptions, accessToken?: string): Promise<Response> {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), options.timeoutMs ?? (options.form ? UPLOAD_TIMEOUT_MS : DEFAULT_TIMEOUT_MS));
    const onAbort = () => controller.abort();
    options.signal?.addEventListener("abort", onAbort, { once: true });

    const headers: Record<string, string> = { Accept: "application/json" };
    if (accessToken) headers.Authorization = `Bearer ${accessToken}`;
    let body: BodyInit | undefined;
    if (options.form) {
      body = options.form; // the runtime sets the multipart boundary
    } else if (options.body !== undefined) {
      headers["Content-Type"] = "application/json";
      body = JSON.stringify(options.body);
    }

    try {
      return await (this.deps.fetchImpl ?? fetch)(`${this.deps.baseUrl}${path}`, {
        method: options.method ?? "GET",
        headers,
        body,
        signal: controller.signal
      });
    } catch (error) {
      if (options.signal?.aborted) throw error; // caller cancelled: not a network failure
      throw new NetworkError();
    } finally {
      clearTimeout(timeout);
      options.signal?.removeEventListener("abort", onAbort);
    }
  }

  /** Single-flight: concurrent 401s share one refresh call, so a rotated refresh token is never reused. */
  private refreshTokens(refreshToken: string | undefined): Promise<AuthTokens | null> {
    if (!this.refreshing) {
      this.refreshing = this.doRefresh(refreshToken).finally(() => {
        this.refreshing = null;
      });
    }
    return this.refreshing;
  }

  /** Resolves to the new tokens, or null when the server rejected the refresh token. Network failures throw. */
  private async doRefresh(refreshToken: string | undefined): Promise<AuthTokens | null> {
    const stored = await this.deps.tokens.load();
    const token = stored?.refreshToken ?? refreshToken;
    if (!token) return null;
    const response = await this.send("/auth/refresh", { method: "POST", body: { refreshToken: token } });
    if (!response.ok) {
      // Only a definitive rejection ends the session; a 5xx or a rate limit must not log the user out.
      if (response.status === 401 || response.status === 400) {
        await this.deps.tokens.clear();
        return null;
      }
      throw new NetworkError();
    }
    const data = (await response.json()) as AuthResponse;
    const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
    await this.deps.tokens.save(next);
    return next;
  }

  private async parse<T>(response: Response): Promise<T> {
    if (response.ok) {
      const text = await response.text();
      return (text ? JSON.parse(text) : undefined) as T;
    }
    let code = "http_error";
    let message = `Erreur ${response.status}`;
    try {
      const payload = (await response.json()) as { error?: string; code?: string };
      if (payload.code) code = payload.code;
      if (payload.error) message = payload.error;
    } catch {
      // Non-JSON error body (proxy page): keep the generic message.
    }
    const retryAfter = Number(response.headers.get("Retry-After"));
    throw new ApiError(response.status, code, message, Number.isFinite(retryAfter) && retryAfter > 0 ? retryAfter : undefined);
  }
}

export const api = new ApiClient({ baseUrl: API_BASE_URL, tokens: tokenStore });
