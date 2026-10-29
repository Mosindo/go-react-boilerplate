import { API_BASE_URL } from "../config";
import { translateApiError } from "./errorMessages";
import { clearTokens, getTokensSync, loadTokens, saveTokens } from "./session";
import type { AuthSession } from "./types";

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

const REQUEST_TIMEOUT_MS = 15_000;

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  query?: Record<string, string | number | undefined | null>;
  formData?: FormData;
  /** Set false for endpoints that must not carry (or refresh) credentials. */
  auth?: boolean;
  signal?: AbortSignal;
};

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  if (!API_BASE_URL) {
    throw new ApiError(0, "L'adresse du serveur n'est pas configurée (EXPO_PUBLIC_API_URL).");
  }
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined && value !== null && value !== "") {
      params.set(key, String(value));
    }
  }
  const qs = params.toString();
  return `${API_BASE_URL}${path}${qs ? `?${qs}` : ""}`;
}

async function parseError(response: Response): Promise<ApiError> {
  let message = `Erreur serveur (${response.status})`;
  try {
    const data: unknown = await response.json();
    if (typeof data === "object" && data !== null && "error" in data) {
      const error = (data as { error: unknown }).error;
      if (typeof error === "string" && error) {
        message = error;
      }
    }
  } catch {
    // Non-JSON error body: keep the generic message.
  }
  if (response.status === 429) {
    message = "Trop de requêtes, patientez un instant avant de réessayer.";
  }
  return new ApiError(response.status, translateApiError(message));
}

async function send(
  path: string,
  options: RequestOptions,
  token: string | null
): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  let body: BodyInit | undefined;
  if (options.formData) {
    body = options.formData;
  } else if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  try {
    return await fetch(buildUrl(path, options.query), {
      method: options.method ?? "GET",
      headers,
      body,
      signal: controller.signal
    });
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    if (controller.signal.aborted && !options.signal?.aborted) {
      throw new ApiError(0, "Le serveur met trop de temps à répondre. Réessayez.");
    }
    throw new ApiError(0, "Connexion impossible. Vérifiez votre réseau et réessayez.");
  } finally {
    clearTimeout(timeout);
  }
}

let refreshInFlight: Promise<string | null> | null = null;

/** Exchanges the refresh token for new tokens. Concurrent callers share one request. */
function refreshAccessToken(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      const current = getTokensSync();
      if (!current?.refreshToken) {
        return null;
      }
      try {
        const response = await send(
          "/auth/refresh",
          { method: "POST", body: { refreshToken: current.refreshToken }, auth: false },
          null
        );
        if (!response.ok) {
          if (response.status === 401) {
            await clearTokens();
          }
          return null;
        }
        const session = (await response.json()) as AuthSession;
        await saveTokens({ accessToken: session.accessToken, refreshToken: session.refreshToken });
        return session.accessToken;
      } catch {
        return null; // transient network failure: keep the session, fail this request
      } finally {
        refreshInFlight = null;
      }
    })();
  }
  return refreshInFlight;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth !== false;
  const tokens = useAuth ? await loadTokens() : null;

  let response = await send(path, options, tokens?.accessToken ?? null);
  if (response.status === 401 && useAuth && tokens) {
    const fresh = await refreshAccessToken();
    if (fresh) {
      response = await send(path, options, fresh);
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

/** Fetches a protected image and returns the headers needed to display it. */
export function authImageSource(url: string): { uri: string; headers: Record<string, string> } {
  const token = getTokensSync()?.accessToken;
  return {
    uri: `${API_BASE_URL}${url}`,
    headers: token ? { Authorization: `Bearer ${token}` } : {}
  };
}
